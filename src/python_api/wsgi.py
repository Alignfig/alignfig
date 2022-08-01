import sys

path = './core/api'
if path not in sys.path:
   sys.path.append(path)

from core.api.app import app as application  # noqa
