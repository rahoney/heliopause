#!/usr/bin/env python3
"""Acquire pinned representative wheels as data; never install or import them."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import tempfile
import urllib.parse
import urllib.request

MANIFEST = Path(__file__).resolve().parent.parent / "internal/artifact/pypi/testdata/real-corpus.json"
MAX_BYTES = 512 * 1024 * 1024


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def checked_url(url, host):
    parsed = urllib.parse.urlsplit(url)
    if parsed.scheme != "https" or parsed.hostname != host or parsed.username or parsed.password or parsed.port:
        raise ValueError("unauthorized corpus URL: " + url)
    return url


class OfficialRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        checked_url(newurl, urllib.parse.urlsplit(req.full_url).hostname)
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def prepare(root, cache):
    opener = urllib.request.build_opener(OfficialRedirects)
    entries = json.loads(MANIFEST.read_text())
    if not entries:
        raise ValueError("empty corpus manifest")
    root.mkdir(parents=True, exist_ok=True)
    for entry in entries:
        alias, filename = entry["fileOnDisk"], entry["canonical"]
        if Path(alias).name != alias or Path(filename).name != filename or entry["source"] != "pypi":
            raise ValueError("invalid corpus identity")
        expected = entry["expectedSHA256"]
        target = root / alias
        if target.exists():
            if digest(target) != expected:
                raise ValueError("existing corpus digest mismatch: " + alias)
            print(alias, expected, "verified existing", flush=True)
            continue
        cached = cache / alias if cache else None
        with tempfile.TemporaryDirectory(prefix="corpus-", dir=root) as work:
            temporary = Path(work) / "wheel"
            if cached and cached.is_file():
                shutil.copyfile(cached, temporary)
                origin = str(cached)
            else:
                index = "https://pypi.org/pypi/" + entry["project"] + "/" + entry["version"] + "/json"
                if entry["index"] != index:
                    raise ValueError("source ownership mismatch")
                with opener.open(checked_url(index, "pypi.org"), timeout=60) as response:
                    metadata = response.read(4 * 1024 * 1024 + 1)
                if len(metadata) > 4 * 1024 * 1024:
                    raise ValueError("index metadata exceeds bound")
                matches = [f for f in json.loads(metadata)["urls"] if f["filename"] == filename]
                if len(matches) != 1 or matches[0]["digests"]["sha256"] != expected:
                    raise ValueError("pinned filename/digest absent from authorized index: " + filename)
                selected = matches[0]
                if not 0 < selected["size"] <= MAX_BYTES:
                    raise ValueError("wheel exceeds acquisition bound")
                origin = checked_url(selected["url"], "files.pythonhosted.org")
                with opener.open(origin, timeout=60) as response, temporary.open("wb") as output:
                    total = 0
                    while block := response.read(1024 * 1024):
                        total += len(block)
                        if total > selected["size"]:
                            raise ValueError("wheel exceeds declared size")
                        output.write(block)
                if total != selected["size"]:
                    raise ValueError("truncated wheel")
            if digest(temporary) != expected:
                raise ValueError("corpus digest mismatch: " + filename)
            temporary.replace(target)
        print(alias, expected, origin, flush=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, required=True)
    parser.add_argument("--cache", type=Path, help="optional acceleration; every hit is rehashed")
    args = parser.parse_args()
    prepare(args.root, args.cache)
