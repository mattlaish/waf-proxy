#!/usr/bin/env python3
from __future__ import annotations
import importlib.util, json, pathlib, sys, tempfile, unittest
ROOT=pathlib.Path(__file__).resolve().parents[3]
P=ROOT/'packaging/cleanhost/clean_host_qualify.py'
spec=importlib.util.spec_from_file_location('clean_host_qualify',P); mod=importlib.util.module_from_spec(spec); sys.modules[spec.name]=mod; spec.loader.exec_module(mod)

class T(unittest.TestCase):
 def test_matrix_complete(self):
  self.assertEqual(set(mod.PLATFORMS), {'debian-12','ubuntu-22.04','ubuntu-24.04','rhel-9','rocky-9','almalinux-9','oraclelinux-9'})
 def test_validate_debian12(self):
  x=mod.validate_platform('debian-12', {'ID':'debian','VERSION_ID':'12.9','PRETTY_NAME':'Debian 12'})
  self.assertEqual(x['id'],'debian')
 def test_wrong_platform_blocked(self):
  with self.assertRaises(mod.Blocked): mod.validate_platform('ubuntu-24.04', {'ID':'ubuntu','VERSION_ID':'22.04'})
 def test_crs_requires_setup_and_rules(self):
  with tempfile.TemporaryDirectory() as td:
   p=pathlib.Path(td); (p/'rules').mkdir(); (p/'crs-setup.conf').write_text('x\n'); (p/'rules/a.conf').write_text('y\n')
   digest,count=mod.validate_crs(p); self.assertRegex(digest,r'^[0-9a-f]{64}$'); self.assertEqual(count,2)
 def test_report_does_not_contain_secret_fields(self):
  with tempfile.TemporaryDirectory() as td:
   class A: platform='debian-12'
   m=mod.PackageMeta('/tmp/a','1','amd64','a'*64)
   r=mod.report_base(A(), {'id':'debian','version_id':'12'}, 'deb', m,m,'b'*64,2,'NOT_REQUIRED')
   text=json.dumps(r).lower(); self.assertNotIn('admin_token',text); self.assertNotIn('token_hash',text)
 def test_preflight_status_is_not_run(self):
  class A: platform='debian-12'
  m=mod.PackageMeta('/tmp/a','1','amd64','a'*64)
  r=mod.report_base(A(), {'id':'debian','version_id':'12'}, 'deb', m,m,'b'*64,2,'NOT_REQUIRED')
  self.assertEqual(r['status'],'NOT_RUN'); self.assertFalse(r['executed'])
 def test_ack_exact(self): self.assertEqual(mod.ACK,'I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_CLEAN_HOST')

if __name__=='__main__': unittest.main()
