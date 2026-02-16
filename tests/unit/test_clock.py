from __future__ import annotations

import re

from scripts.core.clock import Clock


def test_now_iso_returns_iso8601_with_timezone() -> None:
    value = Clock().now_iso()

    assert "T" in value
    assert re.search(r"[+-]\d{2}:\d{2}$", value) is not None


def test_timestamp_matches_expected_format() -> None:
    value = Clock().timestamp()

    assert re.fullmatch(r"\d{8}-\d{6}", value) is not None
