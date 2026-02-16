from __future__ import annotations

from abc import ABC, abstractmethod


class Command(ABC):
    @abstractmethod
    def run(self) -> int:
        raise NotImplementedError
