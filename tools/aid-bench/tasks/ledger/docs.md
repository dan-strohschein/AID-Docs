# ledger — Double-Entry Bookkeeping

A double-entry bookkeeping system. Every financial transaction is recorded as balanced entries across accounts.

## Core Concepts

**Double-entry rule:** Every transaction must have debit entries that equal credit entries. Money doesn't appear or disappear — it moves between accounts.

**Account types and normal balances:** Each account type has a "normal balance" direction:
- Asset and Expense accounts: normal balance is DEBIT (debits increase, credits decrease)
- Liability, Revenue, and Equity accounts: normal balance is CREDIT (credits increase, debits decrease)

The `balance()` method returns the balance in the normal direction. A $500 credit to a Revenue account shows as a positive balance of 500.

## Creating a Ledger

```python
from ledger import Ledger, AccountType, EntryType

ledger = Ledger()
cash = ledger.create_account("Cash", AccountType.ASSET, "1000")
revenue = ledger.create_account("Revenue", AccountType.REVENUE, "4000")
```

## Posting Transactions

Transactions follow a two-phase workflow: **create → verify → post**.

```python
txn = ledger.create_transaction("2024-01-15", memo="Product sale")
txn.add_entry(cash, 500.0, EntryType.DEBIT)
txn.add_entry(revenue, 500.0, EntryType.CREDIT)

result = txn.verify()  # Returns VerifyResult, does NOT raise
if result.ok:
    txn.post()  # Raises if not verified, already posted, or period closed
```

**Important:** `verify()` does not raise exceptions — it returns a `VerifyResult` with `.ok`, `.warnings`, and `.errors` fields. But `post()` DOES raise if:
- Transaction is not verified (`UnverifiedError`)
- Transaction is already posted (`AlreadyPostedError`)
- Transaction date falls in a closed period (`PeriodClosedError`)

## Period Closing

```python
ledger.close_period("2024-01")  # Closes January 2024
```

After closing, you can still create and verify transactions for that period, but `post()` will raise `PeriodClosedError`.

## Querying Balances

```python
balance = cash.balance()                    # Current total balance
balance = cash.balance(as_of="2024-01-31")  # Balance as of a specific date (inclusive)
```

`balance()` computes the running balance from all posted transactions. It only includes posted transactions (not draft or void).

## Trial Balance

```python
balances = ledger.trial_balance()           # dict of {account_name: balance}
balances = ledger.trial_balance(as_of="2024-01-31")
```

## Error Types

- `LedgerError` — base error class
- `UnverifiedError` — tried to post without verifying
- `AlreadyPostedError` — tried to post or modify an already-posted transaction
- `PeriodClosedError` — tried to post to a closed period
- `UnbalancedError` — transaction debits don't equal credits
- `IntegrityError` — trial balance doesn't balance (internal error)
