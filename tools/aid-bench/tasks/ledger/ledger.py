"""Double-entry bookkeeping ledger — synthetic library for AID benchmarking."""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from typing import Optional


class AccountType(Enum):
    ASSET = "asset"
    LIABILITY = "liability"
    EQUITY = "equity"
    REVENUE = "revenue"
    EXPENSE = "expense"


class EntryType(Enum):
    DEBIT = "debit"
    CREDIT = "credit"


class TransactionState(Enum):
    DRAFT = "draft"
    VERIFIED = "verified"
    POSTED = "posted"
    VOID = "void"


# Normal balance direction: which entry type increases the account
_NORMAL_BALANCE = {
    AccountType.ASSET: EntryType.DEBIT,
    AccountType.EXPENSE: EntryType.DEBIT,
    AccountType.LIABILITY: EntryType.CREDIT,
    AccountType.REVENUE: EntryType.CREDIT,
    AccountType.EQUITY: EntryType.CREDIT,
}


class LedgerError(Exception):
    pass


class UnverifiedError(LedgerError):
    pass


class AlreadyPostedError(LedgerError):
    pass


class PeriodClosedError(LedgerError):
    pass


class UnbalancedError(LedgerError):
    pass


class IntegrityError(LedgerError):
    pass


@dataclass
class Entry:
    account: Account
    amount: float
    entry_type: EntryType
    transaction: Transaction


@dataclass
class VerifyResult:
    ok: bool
    warnings: list[str] = field(default_factory=list)
    errors: list[str] = field(default_factory=list)


class Account:
    def __init__(self, name: str, account_type: AccountType, number: str, ledger: Ledger):
        self.name = name
        self.account_type = account_type
        self.number = number
        self._ledger = ledger

    def balance(self, as_of: str | None = None) -> float:
        """Compute the running balance for this account.

        Returns the balance in the account's normal direction:
        - Asset/Expense: debits increase, credits decrease
        - Liability/Revenue/Equity: credits increase, debits decrease

        A positive balance means the account has its expected direction.
        """
        normal = _NORMAL_BALANCE[self.account_type]
        total = 0.0

        for txn in self._ledger._transactions:
            if txn.state != TransactionState.POSTED:
                continue
            if as_of is not None and txn.date > as_of:
                continue
            for entry in txn.entries:
                if entry.account is not self:
                    continue
                if entry.entry_type == normal:
                    total += entry.amount
                else:
                    total -= entry.amount

        return total

    def entries(self, start: str | None = None, end: str | None = None) -> list[Entry]:
        """Get all posted entries for this account, optionally filtered by date range."""
        result = []
        for txn in self._ledger._transactions:
            if txn.state != TransactionState.POSTED:
                continue
            if start is not None and txn.date < start:
                continue
            if end is not None and txn.date > end:
                continue
            for entry in txn.entries:
                if entry.account is self:
                    result.append(entry)
        return result

    def __repr__(self) -> str:
        return f"Account({self.name!r}, {self.account_type.value}, {self.number})"


class Transaction:
    def __init__(self, date: str, memo: str | None, ledger: Ledger):
        self.date = date
        self.memo = memo
        self.state = TransactionState.DRAFT
        self.entries: list[Entry] = []
        self._ledger = ledger

    def add_entry(self, account: Account, amount: float, entry_type: str | EntryType) -> None:
        """Add an entry to this transaction. Does not validate balance."""
        if self.state in (TransactionState.POSTED, TransactionState.VOID):
            raise AlreadyPostedError("Cannot modify a posted or voided transaction")

        if isinstance(entry_type, str):
            entry_type = EntryType(entry_type)

        if amount <= 0:
            raise LedgerError("Entry amount must be positive")

        self.entries.append(Entry(
            account=account,
            amount=amount,
            entry_type=entry_type,
            transaction=self,
        ))
        # If it was verified, adding an entry resets to draft
        if self.state == TransactionState.VERIFIED:
            self.state = TransactionState.DRAFT

    def verify(self) -> VerifyResult:
        """Verify the transaction is ready to post.

        Returns a VerifyResult — does NOT raise on failure.
        Sets state to VERIFIED if no errors.
        """
        warnings = []
        errors = []

        if not self.entries:
            errors.append("Transaction has no entries")

        if self.memo is None or self.memo.strip() == "":
            warnings.append("Transaction has no memo")

        total_debits = sum(e.amount for e in self.entries if e.entry_type == EntryType.DEBIT)
        total_credits = sum(e.amount for e in self.entries if e.entry_type == EntryType.CREDIT)

        if abs(total_debits - total_credits) > 0.001:
            errors.append(
                f"Transaction is unbalanced: debits={total_debits:.2f}, credits={total_credits:.2f}"
            )

        ok = len(errors) == 0

        if ok:
            self.state = TransactionState.VERIFIED

        return VerifyResult(ok=ok, warnings=warnings, errors=errors)

    def post(self) -> None:
        """Post the transaction to the ledger.

        Raises UnverifiedError if not verified.
        Raises AlreadyPostedError if already posted.
        Raises PeriodClosedError if the transaction date falls in a closed period.
        """
        if self.state == TransactionState.DRAFT:
            raise UnverifiedError("Transaction must be verified before posting")

        if self.state == TransactionState.POSTED:
            raise AlreadyPostedError("Transaction is already posted")

        if self.state == TransactionState.VOID:
            raise AlreadyPostedError("Cannot post a voided transaction")

        # Check period closure
        period = self.date[:7]  # "YYYY-MM"
        if period in self._ledger._closed_periods:
            raise PeriodClosedError(f"Period {period} is closed")

        self.state = TransactionState.POSTED

    def void(self) -> None:
        """Void a posted transaction by creating reverse entries.

        Only posted transactions can be voided.
        """
        if self.state != TransactionState.POSTED:
            raise LedgerError("Only posted transactions can be voided")

        # Create a reversing transaction
        reverse = self._ledger.create_transaction(
            date=self.date,
            memo=f"VOID: {self.memo or 'no memo'}"
        )
        for entry in self.entries:
            # Reverse the entry type
            rev_type = EntryType.CREDIT if entry.entry_type == EntryType.DEBIT else EntryType.DEBIT
            reverse.add_entry(entry.account, entry.amount, rev_type)

        result = reverse.verify()
        if result.ok:
            reverse.post()

        self.state = TransactionState.VOID

    def __repr__(self) -> str:
        return f"Transaction({self.date!r}, state={self.state.value}, entries={len(self.entries)})"


class Ledger:
    def __init__(self):
        self._accounts: dict[str, Account] = {}
        self._accounts_by_number: dict[str, Account] = {}
        self._transactions: list[Transaction] = []
        self._closed_periods: set[str] = set()

    def create_account(self, name: str, account_type: str | AccountType, number: str) -> Account:
        """Create a new account in the ledger."""
        if name in self._accounts:
            raise LedgerError(f"Account '{name}' already exists")
        if number in self._accounts_by_number:
            raise LedgerError(f"Account number '{number}' already in use")

        if isinstance(account_type, str):
            account_type = AccountType(account_type)

        account = Account(name, account_type, number, self)
        self._accounts[name] = account
        self._accounts_by_number[number] = account
        return account

    def get_account(self, name_or_number: str) -> Account:
        """Look up an account by name or number."""
        if name_or_number in self._accounts:
            return self._accounts[name_or_number]
        if name_or_number in self._accounts_by_number:
            return self._accounts_by_number[name_or_number]
        raise LedgerError(f"Account not found: {name_or_number}")

    def create_transaction(self, date: str, memo: str | None = None) -> Transaction:
        """Create a new draft transaction."""
        txn = Transaction(date, memo, self)
        self._transactions.append(txn)
        return txn

    def close_period(self, period: str) -> None:
        """Close a period (YYYY-MM format). No new transactions can be posted to closed periods."""
        self._closed_periods.add(period)

    def trial_balance(self, as_of: str | None = None) -> dict[str, float]:
        """Compute trial balance — all account balances.

        Returns a dict of account_name -> balance (in normal direction).
        Raises IntegrityError if total debits != total credits (should never happen).
        """
        balances = {}
        total_debit_normal = 0.0
        total_credit_normal = 0.0

        for name, account in self._accounts.items():
            bal = account.balance(as_of=as_of)
            balances[name] = bal

            if _NORMAL_BALANCE[account.account_type] == EntryType.DEBIT:
                total_debit_normal += bal
            else:
                total_credit_normal += bal

        if abs(total_debit_normal - total_credit_normal) > 0.01:
            raise IntegrityError(
                f"Trial balance does not balance: "
                f"debit-normal={total_debit_normal:.2f}, credit-normal={total_credit_normal:.2f}"
            )

        return balances
