# ledger — Available Classes and Methods

Ledger()
Ledger.create_account(name, account_type, number)
Ledger.get_account(name_or_number)
Ledger.create_transaction(date, memo)
Ledger.close_period(period)
Ledger.trial_balance(as_of)

Account.name
Account.account_type
Account.number
Account.balance(as_of)
Account.entries(start, end)

Transaction.date
Transaction.memo
Transaction.state
Transaction.entries
Transaction.add_entry(account, amount, entry_type)
Transaction.verify()
Transaction.post()
Transaction.void()

AccountType.ASSET
AccountType.LIABILITY
AccountType.EQUITY
AccountType.REVENUE
AccountType.EXPENSE

EntryType.DEBIT
EntryType.CREDIT

TransactionState.DRAFT
TransactionState.VERIFIED
TransactionState.POSTED
TransactionState.VOID

VerifyResult.ok
VerifyResult.warnings
VerifyResult.errors
