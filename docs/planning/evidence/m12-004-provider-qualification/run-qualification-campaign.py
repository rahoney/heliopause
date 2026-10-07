from pathlib import Path
import subprocess,json,datetime,os,shutil,hashlib
r=Path('/home/winamp/.local/state/heliopause/m12-004');q=r/'qualification'; driver=r/'run-qualified-consumers.sh'
root=Path('/tmp/haa-m12-004-native-e2e')
assert not root.exists(), 'fresh retained evidence root already exists'
working=3*3656681374+6556756381
free=os.statvfs('/mnt/c').f_bavail*os.statvfs('/mnt/c').f_frsize
assert free>=working+8*(1<<30)
root.mkdir(mode=0o700)
(q/'campaign-capacity.json').write_text(json.dumps({'physical_C_free':free,'case_working_set':working,'reserve':8*(1<<30),'fresh_evidence_root':str(root),'cache':'fresh per-case native roots'},indent=2)+'\n')
steps=[
 ('terraform-boundary-final','^TestLinuxTerraformProvider(Probe|Security)Integration$','none','5m','sandbox'),
 ('legacy-direct','^TestLinux(GVisorLifecycle|GitHubReleaseELFDynamic|PyPISdistBuild|PyPIWheelDynamic|PythonRootPolicyTransactionSequence|PythonRenamedProcessTransaction|PythonPinnedLibutilTransaction|RetainedPythonTransaction|GoIsolatedResolver|GoIsolatedProjectResolver|GoSourceProjectSnapshot|GoDependencyFreeProjectSnapshot|CargoSourceProjectSnapshot)Integration$','none','15m','sandbox'),
 ('python-proc-maps-cpu','^TestLinuxPythonProcMapsTransactionIntegration$','none','5m','sandbox'),
 ('python-proc-maps-cu126','^TestLinuxPythonProcMapsTransactionIntegration$','cu126','5m','sandbox'),
 ('python-proc-maps-cu130','^TestLinuxPythonProcMapsTransactionIntegration$','cu130','5m','sandbox'),
 ('python-proc-maps-cu132','^TestLinuxPythonProcMapsTransactionIntegration$','cu132','5m','sandbox'),
 ('ordinary-product-cli',None,None,None,None),
 ('go-cargo-cli','^TestLinuxGo(GetDownload|TransitiveGetDownload|DependencyFreeDownload|DependencyFreeBuildCLI|PublicBuildCLI|TransitiveBuildCLI|InvalidTestdataBuildCLI|CgoBuildCLI|LibraryBuildCLI|MixedPackagesBuildCLI|MixedInvalidPackageBuildCLI|BuildSecurityCLI)Integration$|^TestLinuxCargo(Add|TransitiveAdd)Integration$','none','30m','bootstrap'),
 ('cargo-build-cli','^TestLinuxCargo.*Build.*CLIIntegration$|^TestLinuxCargoProcMacroSecurityCLIIntegration$','none','15m','bootstrap'),
 ('promotion-consumers','^TestLinux(NPMPromotion|PyPIPromotion)Integration$','none','5m','promotion'),
 ('cpu-full','^TestLinuxPyTorchFullIntegration$','cpu','15m','bootstrap'),
 ('cu126-full','^TestLinuxPyTorchFullIntegration$','cu126','40m','bootstrap'),
]
# The final Terraform tests require their explicit opt-in on the installed client.
p=driver;s=p.read_text().replace('HELOX_GO_BUILD_CLI_INTEGRATION=1','HELOX_TERRAFORM_PROVIDER_INTEGRATION=1 HELOX_GO_BUILD_CLI_INTEGRATION=1');p.write_text(s)
status=[]
for name,regex,profile,timeout,package in steps:
 cmd=['bash',str(r/'run-qualified-cli.sh')] if regex is None else ['bash',str(driver),regex,profile,timeout,package]
 record={'name':name,'command':cmd,'status':'RUNNING','started_utc':datetime.datetime.now(datetime.timezone.utc).isoformat()};status.append(record)
 (q/'campaign.json').write_text(json.dumps(status,indent=2)+'\n')
 log=q/(name+'-runtime.log')
 with log.open('xb') as f: result=subprocess.run(cmd,stdout=f,stderr=subprocess.STDOUT,cwd='/tmp/haa-m12-004-source')
 record.update(status='PASS' if result.returncode==0 else 'FAIL',exit_code=result.returncode,finished_utc=datetime.datetime.now(datetime.timezone.utc).isoformat(),raw_log=log.name,raw_sha256=hashlib.sha256(log.read_bytes()).hexdigest())
 custody=q/(name+'-custody');custody.mkdir(mode=0o700)
 for pattern in ['*-after.modes','*.log','*.pid','*.wait','helper-launch*','helper-stop*']:
  for p in (r/'lifecycle').glob(pattern):
   if p.is_file() and p.stat().st_size<4*1024*1024:shutil.copyfile(p,custody/p.name)
 (q/'campaign.json').write_text(json.dumps(status,indent=2)+'\n')
 print(name,record['status'],result.returncode,flush=True)
 if result.returncode:raise SystemExit(result.returncode)
