"""Module with async functions."""


async def fetch(url: str) -> bytes:
    """Fetch bytes from a URL."""
    ...


async def send(url: str, data: bytes, retries: int = 3) -> bool:
    """Send data to a URL."""
    ...


def sync_helper(x: int) -> int:
    """A regular sync function alongside async ones."""
    ...
