from __future__ import annotations

from pathlib import Path

import pytest

from scripts.core.file_scanner import FileScanner


def test_find_large_files_sorts_desc_and_limits(tmp_path: Path) -> None:
    notes = tmp_path / "notes"
    repos = tmp_path / "repos"
    notes.mkdir()
    repos.mkdir()

    (notes / "small.txt").write_bytes(b"a" * 10)
    (notes / "big.txt").write_bytes(b"b" * 300)
    (repos / "bigger.txt").write_bytes(b"c" * 500)

    scanner = FileScanner()
    results = scanner.find_large_files(
        [notes, repos], threshold_bytes=50, limit=2)

    assert len(results) == 2
    assert results[0][0] == 500
    assert results[1][0] == 300


def test_find_large_files_ignores_missing_paths(tmp_path: Path) -> None:
    scanner = FileScanner()
    missing = tmp_path / "missing"

    results = scanner.find_large_files([missing], threshold_bytes=10)

    assert results == []


def test_find_hotspots_counts_and_limits(tmp_path: Path) -> None:
    notes = tmp_path / "notes"
    repos = tmp_path / "repos"
    notes_deep = notes / "deep"
    repos_deep = repos / "deep"
    notes_deep.mkdir(parents=True)
    repos_deep.mkdir(parents=True)

    for index in range(5):
        (notes_deep / f"n-{index}.txt").write_text("n", encoding="utf-8")
    for index in range(3):
        (repos_deep / f"r-{index}.txt").write_text("r", encoding="utf-8")

    scanner = FileScanner()
    results = scanner.find_hotspots([notes, repos], threshold=2, limit=1)

    assert len(results) == 1
    assert results[0][0] == 5
    assert results[0][1].endswith("notes/deep")


def test_find_hotspots_returns_empty_when_no_matches(tmp_path: Path) -> None:
    notes = tmp_path / "notes"
    notes.mkdir()
    (notes / "a.txt").write_text("a", encoding="utf-8")

    scanner = FileScanner()
    results = scanner.find_hotspots([notes], threshold=5)

    assert results == []


def test_find_large_files_skips_oserror_stat(
    tmp_path: Path,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    notes = tmp_path / "notes"
    notes.mkdir()
    (notes / "good.txt").write_bytes(b"g" * 200)
    (notes / "bad.txt").write_bytes(b"b" * 200)

    original_stat = Path.stat

    def patched_stat(self: Path):
        if self.name == "bad.txt":
            raise OSError("simulated stat failure")
        return original_stat(self)

    monkeypatch.setattr(Path, "stat", patched_stat)

    results = FileScanner().find_large_files([notes], threshold_bytes=100)

    assert len(results) == 1
    assert results[0][1].endswith("good.txt")


def test_find_hotspots_ignores_missing_paths(tmp_path: Path) -> None:
    missing = tmp_path / "missing"

    results = FileScanner().find_hotspots([missing], threshold=1)

    assert results == []
