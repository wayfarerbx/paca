"""Root conftest: set required env vars before any test module is imported.

src/config.py calls Settings() at module level, so DATABASE_URL and
INTERNAL_API_KEY must be present in the environment before collection starts.
"""

import os

os.environ.setdefault("DATABASE_URL", "postgresql://test:test@localhost/test")
os.environ.setdefault("INTERNAL_API_KEY", "test-internal-key")
