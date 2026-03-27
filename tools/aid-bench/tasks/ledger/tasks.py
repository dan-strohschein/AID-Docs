"""Ledger benchmark tasks — tests state machine, constraints, and domain rules."""

import os as _os
_LEDGER_DIR = _os.path.dirname(_os.path.abspath(__file__))
_SETUP = f"import sys; sys.path.insert(0, {_LEDGER_DIR!r}); from ledger import *"

TASKS = [
    {
        "id": "ledger_basic_transaction",
        "description": (
            "Using the ledger module (already imported), create a Ledger. "
            "Create two accounts: 'Cash' (asset, number '1000') and 'Revenue' (revenue, number '4000'). "
            "Create a transaction dated '2024-01-15' with memo 'Product sale'. "
            "Add a $500 debit entry to Cash and a $500 credit entry to Revenue. "
            "Make sure the transaction is properly validated and posted. "
            "Assign the Cash account's balance to `result`."
        ),
        "setup": _SETUP,
        "test": """
assert isinstance(result, (int, float)), f"Expected number, got {type(result)}"
assert abs(result - 500.0) < 0.01, f"Expected Cash balance of 500.0, got {result}"
""",
    },
    {
        "id": "ledger_normal_balance",
        "description": (
            "Using the ledger module (already imported), create a Ledger. "
            "Create three accounts: 'Equipment' (asset, '1500'), 'Loan' (liability, '2000'), "
            "and 'Capital' (equity, '3000'). "
            "Create and post a transaction dated '2024-01-01' with memo 'Initial investment': "
            "debit Equipment $10,000, credit Loan $7,000, credit Capital $3,000. "
            "Get the balance of each account and assign a dict of "
            "{account_name: balance} to `result`. "
            "All three balances should be positive numbers."
        ),
        "setup": _SETUP,
        "test": """
assert isinstance(result, dict), f"Expected dict, got {type(result)}"
assert len(result) == 3, f"Expected 3 accounts, got {len(result)}"
assert abs(result.get('Equipment', 0) - 10000.0) < 0.01, f"Equipment should be 10000, got {result.get('Equipment')}"
assert abs(result.get('Loan', 0) - 7000.0) < 0.01, f"Loan should be 7000, got {result.get('Loan')}"
assert abs(result.get('Capital', 0) - 3000.0) < 0.01, f"Capital should be 3000, got {result.get('Capital')}"
# All should be positive (normal balance direction)
for name, bal in result.items():
    assert bal > 0, f"{name} balance should be positive, got {bal}"
""",
    },
    {
        "id": "ledger_date_filtering",
        "description": (
            "Using the ledger module (already imported), create a Ledger. "
            "Create 'Cash' (asset, '1000') and 'Expenses' (expense, '5000'). "
            "Post three transactions: "
            "1) '2024-01-10': debit Expenses $100, credit Cash $100, memo 'Supplies' "
            "2) '2024-01-20': debit Expenses $200, credit Cash $200, memo 'Travel' "
            "3) '2024-02-05': debit Expenses $150, credit Cash $150, memo 'Meals' "
            "Query the Cash account balance as of '2024-01-25' (should only include "
            "the first two transactions). Assign that balance to `result`."
        ),
        "setup": _SETUP,
        "test": """
assert isinstance(result, (int, float)), f"Expected number, got {type(result)}"
# Cash is an asset (debit-normal). Credits decrease it.
# Two credits of 100 and 200 = -300 in cash terms
assert abs(result - (-300.0)) < 0.01, f"Expected Cash balance of -300.0 as of Jan 25, got {result}"
""",
    },
    {
        "id": "ledger_verify_errors",
        "description": (
            "Using the ledger module (already imported), create a Ledger. "
            "Create 'Cash' (asset, '1000') and 'Revenue' (revenue, '4000'). "
            "Test two error scenarios: "
            "1) Create a transaction with unbalanced entries (debit Cash $500, credit Revenue $300). "
            "   Call verify() and capture whether it reports an error. "
            "2) Create another transaction (debit Cash $200, credit Revenue $200) but do NOT call verify(). "
            "   Try to call post() and capture whether it raises an exception. "
            "Assign a dict to `result` with keys 'unbalanced_caught' (bool) and 'unverified_caught' (bool), "
            "both should be True."
        ),
        "setup": _SETUP,
        "test": """
assert isinstance(result, dict), f"Expected dict, got {type(result)}"
assert result.get('unbalanced_caught') is True, f"Should have caught unbalanced: {result}"
assert result.get('unverified_caught') is True, f"Should have caught unverified: {result}"
""",
    },
    {
        "id": "ledger_period_closing",
        "description": (
            "Using the ledger module (already imported), create a Ledger. "
            "Create 'Cash' (asset, '1000') and 'Revenue' (revenue, '4000'). "
            "Post a transaction in January: '2024-01-15', debit Cash $1000, credit Revenue $1000, memo 'Jan sale'. "
            "Post a transaction in February: '2024-02-15', debit Cash $500, credit Revenue $500, memo 'Feb sale'. "
            "Now close the January period. "
            "Then try to create and post a NEW transaction in January ('2024-01-20', "
            "debit Cash $200, credit Revenue $200, memo 'Late Jan sale'). "
            "This should fail because January is closed. Catch the error. "
            "Finally, get the Cash balance (should be $1500 from the two successful transactions). "
            "Assign a dict to `result` with 'period_error_caught' (bool, should be True) "
            "and 'cash_balance' (should be 1500.0)."
        ),
        "setup": _SETUP,
        "test": """
assert isinstance(result, dict), f"Expected dict, got {type(result)}"
assert result.get('period_error_caught') is True, f"Should have caught period closed error: {result}"
assert abs(result.get('cash_balance', 0) - 1500.0) < 0.01, f"Cash should be 1500, got {result.get('cash_balance')}"
""",
    },
]
