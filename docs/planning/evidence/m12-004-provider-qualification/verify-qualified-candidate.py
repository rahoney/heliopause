from pathlib import Path
import hashlib,json,stat
root=Path('/home/winamp/.local/state/heliopause/m12-004');src=Path('/mnt/c/Users/GPC/OneDrive/06_Code-sandbox/Heliopause-Artifact-Airlock');native=Path('/tmp/haa-m12-004-source')
v=json.loads((root/'qualification/final-executable-candidate.json').read_text())
for record in v['source']:
 for base in [src,native]:
  p=base/record['path'];assert hashlib.sha256(p.read_bytes()).hexdigest()==record['sha256'],str(p)
for record in v['binaries']:
 p=Path(record['path']);s=p.lstat();assert stat.S_ISREG(s.st_mode) and s.st_nlink==1 and ("mode" not in record or stat.S_IMODE(s.st_mode)==record["mode"]) and hashlib.sha256(p.read_bytes()).hexdigest()==record['sha256'],str(p)
print('Exact source/binary probe candidate verified')
