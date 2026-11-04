#!/bin/bash
set -euo pipefail
umask 077

# Only the final native bytes use stdout. All source/build diagnostics remain
# separate, and neither stream grants runtime admission.
exec 3>&1 1>&2
export GIT_CONFIG_NOSYSTEM=1 GIT_CONFIG_GLOBAL=/work/frozen-gitconfig
export GIT_TERMINAL_PROMPT=0 GIT_ALLOW_PROTOCOL=file
export OPAMROOT=/work/opam OPAMSWITCH=s04-review OPAMJOBS=2
export OPAMDOWNLOADJOBS=2 OPAMSOLVERTIMEOUT=300 MAKEFLAGS=-j2
export LIBRARY_PATH=/work/static-curl/lib
export PKG_CONFIG_PATH=/work/static-curl/lib/pkgconfig
export XDG_CACHE_HOME=/work/build-cache

mkdir -p /work/input
tar -xf /source/semgrep-SOURCES.tar -C /work/input --no-same-owner
cd /work/input
# Python is already part of the fixed SDK. Verify bytes and project the bounded
# metadata once; do not add a second JSON tool or a floating SDK dependency.
python3 - <<'PYTHON'
from pathlib import Path
import hashlib,json,os,shlex,subprocess
root=Path('/work/input')
manifest=json.loads((root/'SOURCE-MANIFEST.json').read_text())
for row in manifest['files']:
    with (root/row['path']).open('rb') as member:
        assert hashlib.file_digest(member,'sha256').hexdigest()==row['sha256'],row['path']
origins=json.loads((root/'SOURCE-ORIGINS.json').read_text())
source=origins['source']
assert manifest['source_commit']==source['commit']
assert str(manifest['source_date_epoch'])==os.environ['SOURCE_DATE_EPOCH']
sdk=json.loads((root/'build/sdk-inputs.json').read_text())
values={'SOURCE_COMMIT':source['commit'],'SOURCE_TAG':source['tag'],
        'SOURCE_TAG_OBJECT':source['tag_object'],'SOURCE_LOCK_SHA256':source['lock_sha256'],
        'OPAM_VERSION':sdk['opam_version']}
for package,key in [('curl','CURL_SOURCE'),('tree-sitter','TREE_SOURCE')]:
    values[key]=next(row['relative_path'] for row in origins['native_archives'] if row['package']==package)
values['MAIN_ORIGIN']=next(row['origin'] for row in origins['git_repositories'] if row['name']=='semgrep')
arrays={'GIT_ORIGINS':[row['origin'] for row in origins['git_repositories']],
        'GIT_REPOSITORIES':[row['relative_path'] for row in origins['git_repositories']],
        'PIN_PACKAGES':[row['package'] for row in source['pins']],
        'PIN_ORIGINS':[row['url'] for row in source['pins']],
        'PIN_COMMITS':[row['commit'] for row in source['pins']]}
# The offer records these Gitlinks as unused by this Linux core build. Resolve
# their actual .gitmodules names; no blanket recursive-fetch bypass is used.
unused=[]
for repo in origins['git_repositories']:
    if not repo['gitlinks']:continue
    entries=subprocess.check_output(['git','--git-dir',str(root/repo['relative_path']),
        'config','--blob',repo['commit']+':.gitmodules','--get-regexp',r'^submodule\..*\.path$'],text=True)
    declared={row['path'] for row in repo['gitlinks']}
    matched=set()
    for line in entries.splitlines():
        key,path=line.split(None,1)
        assert path in declared,path
        matched.add(path)
        unused.append(key[:-len('.path')]+'.update')
    assert matched==declared
arrays['UNUSED_SUBMODULE_KEYS']=unused
with Path('/work/build-inputs.sh').open('w') as output:
    for key,value in values.items():output.write(key+'='+shlex.quote(value)+'\n')
    for key,value in arrays.items():output.write(key+'=('+' '.join(map(shlex.quote,value))+')\n')
with Path('/work/cache-inputs.tsv').open('w') as output:
    for row in origins['opam_archives']+origins['opam_extra_sources']:
        for checksum in row['checksums']:output.write(row['relative_path']+'\t'+checksum+'\n')
PYTHON
source /work/build-inputs.sh
test "$(opam --version)" = "$OPAM_VERSION"
printf 'semgrep_build_phase=source_inputs_verified\n'

for i in "${!GIT_ORIGINS[@]}"; do
  git config --file /work/frozen-gitconfig "url.file:///work/input/${GIT_REPOSITORIES[$i]}.insteadOf" "${GIT_ORIGINS[$i]}"
done
for key in "${UNUSED_SUBMODULE_KEYS[@]}"; do
  git config --file /work/frozen-gitconfig "$key" none
done
git clone --no-hardlinks /work/input/git/semgrep.git /work/source
git -C /work/source checkout --detach "$SOURCE_COMMIT"
git -C /work/source remote set-url origin "$MAIN_ORIGIN"
test "$(git -C /work/source rev-parse HEAD)" = "$SOURCE_COMMIT"
test "$(git -C /work/source rev-parse "refs/tags/$SOURCE_TAG")" = "$SOURCE_TAG_OBJECT"
git -C /work/source apply --check /work/input/patches/quarantine.patch
git -C /work/source apply /work/input/patches/quarantine.patch
# The source offer carries regenerated source tables, not a new build tool.
# Every member was authenticated above. Small grammar/converter changes remain
# in the selected patch; generated C/CST stay out of the repository diff.
for language in typescript tsx; do
  for member in parser.c CST.ml Parse.ml Boilerplate.ml; do
    cp "/work/input/generated/$language/$member" "/work/source/languages/typescript/tree-sitter/semgrep-$language/lib/$member"
  done
done
cp /work/source/opam-lockfiles/semgrep.opam.linux-amd64.locked /work/source/semgrep.opam.locked
printf '%s  %s\n' "$SOURCE_LOCK_SHA256" /work/source/semgrep.opam.locked | sha256sum -c -

mkdir /work/curl-source
tar -xf "/work/input/$CURL_SOURCE" -C /work/curl-source --strip-components=1
cd /work/curl-source
./configure --prefix=/work/static-curl --disable-shared --with-ssl --disable-ldap \
  --without-zstd --without-brotli --without-nghttp2 --without-nghttp3 \
  --without-libidn2 --without-libpsl --without-quiche --disable-ares
make -j2
make install

printf 'semgrep_build_phase=fresh_offline_opam\n'
opam init --bare --disable-sandboxing --no-setup --yes default /work/input/opam-repository
opam switch create s04-review --empty --yes

# Seed every declared archive checksum from the already verified source data.
# All aliases are within this disposable tmpfs; no source download is allowed.
while IFS=$'\t' read -r source checksum; do
  algorithm=${checksum%%=*}
  digest=${checksum#*=}
  destination="/work/opam/download-cache/$algorithm/${digest:0:2}/$digest"
  mkdir -p "$(dirname "$destination")"
  if test -e "$destination"; then
    cmp "/work/input/$source" "$destination"
  else
    ln "/work/input/$source" "$destination"
  fi
done < /work/cache-inputs.tsv

# OPAM's full fetch does not itself prefill a fresh client's shallow boundary.
for i in "${!PIN_PACKAGES[@]}"; do
  package=${PIN_PACKAGES[$i]}
  origin=${PIN_ORIGINS[$i]}
  commit=${PIN_COMMITS[$i]}
  name=${package%%.*}
  source="/work/opam/s04-review/.opam-switch/sources/$name"
  mkdir -p "$source"
  git -C "$source" init -q
  git -C "$source" remote add origin "$origin"
  git -C "$source" fetch --depth=1 --update-shallow --no-tags origin "+$commit:refs/remotes/opam-ref-$commit"
done

cd /work/source
tree_version=$(cat libs/ocaml-tree-sitter-semgrep/core/tree-sitter-version)
cp "/work/input/$TREE_SOURCE" "libs/ocaml-tree-sitter-semgrep/core/v${tree_version}.tar.gz"
opam install --skip-updates --deps-only --locked --no-depexts --update-invariant --yes ./semgrep.opam
./scripts/validate-compiler-sha.sh
printf 'semgrep_build_phase=core\n'
opam exec -- bash -c 'set -euo pipefail; source ./tree-sitter-config.sh; dune build -j2 libs/ocaml-tree-sitter-semgrep/core/tree-sitter-out; dune build -j2 _build/install/default/bin/semgrep-core'
strip --strip-unneeded -o /work/semgrep-core _build/install/default/bin/semgrep-core
printf 'semgrep_build_phase=native_ready\n'
cat /work/semgrep-core >&3
