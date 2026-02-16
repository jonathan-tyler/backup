from __future__ import annotations

import re


class SizeParser:
    _pattern = re.compile(r"^(\d+)([KMGTP]?)(B)?$", re.IGNORECASE)

    @classmethod
    def parse_bytes(cls, value: str) -> int:
        match = cls._pattern.match(value.strip())
        if not match:
            raise SystemExit(f"Invalid size format: {value}")
        number = int(match.group(1))
        suffix = match.group(2).upper()
        scale = {
            "": 1,
            "K": 1024,
            "M": 1024**2,
            "G": 1024**3,
            "T": 1024**4,
            "P": 1024**5,
        }[suffix]
        return number * scale
