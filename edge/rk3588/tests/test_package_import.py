import os
import sys
import unittest


RK3588_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
if RK3588_ROOT not in sys.path:
    sys.path.insert(0, RK3588_ROOT)


class PackageImportTest(unittest.TestCase):
    def test_app_supports_package_import(self):
        from face_app.app import RK3588EdgeApplication

        self.assertTrue(callable(RK3588EdgeApplication))


if __name__ == "__main__":
    unittest.main()
