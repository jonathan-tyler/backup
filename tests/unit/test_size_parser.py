from __future__ import annotations

import pytest

from scripts.core.size_parser import SizeParser


@pytest.mark.parametrize(
    ("raw", "expected"),
    [
        ("1", 1),
        ("1K", 1024),
        ("2KB", 2 * 1024),
        ("3M", 3 * 1024**2),
        ("4G", 4 * 1024**3),
        ("5T", 5 * 1024**4),
        ("6P", 6 * 1024**5),
        (" 7m ", 7 * 1024**2),
    ],
)
def test_parse_bytes_valid(raw: str, expected: int) -> None:
    assert SizeParser.parse_bytes(raw) == expected


@pytest.mark.parametrize("raw", ["", "-1G", "1X", "G", "1.5G", "abc"])
def test_parse_bytes_invalid(raw: str) -> None:
    with pytest.raises(SystemExit, match="Invalid size format"):
        SizeParser.parse_bytes(raw)
