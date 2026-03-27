"""A simple HTTP utility module for testing."""

MAX_RETRIES = 3
DEFAULT_TIMEOUT = 30.0
BASE_URL: str = "https://api.example.com"


def get(url: str, timeout: float = 30.0) -> dict:
    """Fetch a resource from the given URL."""
    ...


def post(url: str, data: dict, timeout: float = 30.0) -> dict:
    """Send data to the given URL."""
    ...


async def fetch_all(urls: list[str]) -> list[dict]:
    """Fetch multiple URLs concurrently."""
    ...


def _internal_helper(x: int) -> None:
    """This should not be extracted."""
    ...
