"""Evaluate ledger benchmark results from subagent-generated code."""
import sys
sys.path.insert(0, "src")

from aid_bench.evaluator import evaluate
from aid_bench.runner import load_tasks

tasks = load_tasks("ledger")

# Code from the blind subagent (returned inline)
BLIND_CODE = [
"""
ledger = Ledger()
cash = ledger.create_account('Cash', AccountType.ASSET, '1000')
revenue = ledger.create_account('Revenue', AccountType.REVENUE, '4000')
txn = ledger.create_transaction('2024-01-15', 'Product sale')
txn.add_entry(cash, 500, EntryType.DEBIT)
txn.add_entry(revenue, 500, EntryType.CREDIT)
txn.verify()
txn.post()
result = cash.balance()
""",
"""
ledger = Ledger()
equipment = ledger.create_account('Equipment', AccountType.ASSET, '1500')
loan = ledger.create_account('Loan', AccountType.LIABILITY, '2000')
capital = ledger.create_account('Capital', AccountType.EQUITY, '3000')
txn = ledger.create_transaction('2024-01-01', 'Initial investment')
txn.add_entry(equipment, 10000, EntryType.DEBIT)
txn.add_entry(loan, 7000, EntryType.CREDIT)
txn.add_entry(capital, 3000, EntryType.CREDIT)
txn.verify()
txn.post()
result = {
    'Equipment': equipment.balance(),
    'Loan': loan.balance(),
    'Capital': capital.balance(),
}
""",
"""
ledger = Ledger()
cash = ledger.create_account('Cash', AccountType.ASSET, '1000')
expenses = ledger.create_account('Expenses', AccountType.EXPENSE, '5000')
for date, amt, memo in [('2024-01-10', 100, 'Supplies'), ('2024-01-20', 200, 'Travel'), ('2024-02-05', 150, 'Meals')]:
    t = ledger.create_transaction(date, memo)
    t.add_entry(expenses, float(amt), EntryType.DEBIT)
    t.add_entry(cash, float(amt), EntryType.CREDIT)
    t.verify()
    t.post()
result = cash.balance(as_of='2024-01-25')
""",
"""
ledger = Ledger()
cash = ledger.create_account('Cash', AccountType.ASSET, '1000')
revenue = ledger.create_account('Revenue', AccountType.REVENUE, '4000')
txn1 = ledger.create_transaction('2024-01-15', 'Unbalanced')
txn1.add_entry(cash, 500, EntryType.DEBIT)
txn1.add_entry(revenue, 300, EntryType.CREDIT)
verify_result = txn1.verify()
unbalanced_caught = not verify_result.ok and len(verify_result.errors) > 0
txn2 = ledger.create_transaction('2024-01-16', 'Unverified')
txn2.add_entry(cash, 200, EntryType.DEBIT)
txn2.add_entry(revenue, 200, EntryType.CREDIT)
unverified_caught = False
try:
    txn2.post()
except UnverifiedError:
    unverified_caught = True
result = {'unbalanced_caught': unbalanced_caught, 'unverified_caught': unverified_caught}
""",
"""
ledger = Ledger()
cash = ledger.create_account('Cash', AccountType.ASSET, '1000')
revenue = ledger.create_account('Revenue', AccountType.REVENUE, '4000')
txn_jan = ledger.create_transaction('2024-01-15', 'Jan sale')
txn_jan.add_entry(cash, 1000, EntryType.DEBIT)
txn_jan.add_entry(revenue, 1000, EntryType.CREDIT)
txn_jan.verify()
txn_jan.post()
txn_feb = ledger.create_transaction('2024-02-15', 'Feb sale')
txn_feb.add_entry(cash, 500, EntryType.DEBIT)
txn_feb.add_entry(revenue, 500, EntryType.CREDIT)
txn_feb.verify()
txn_feb.post()
ledger.close_period('2024-01')
period_error_caught = False
try:
    txn_late = ledger.create_transaction('2024-01-20', 'Late Jan sale')
    txn_late.add_entry(cash, 200, EntryType.DEBIT)
    txn_late.add_entry(revenue, 200, EntryType.CREDIT)
    txn_late.verify()
    txn_late.post()
except PeriodClosedError:
    period_error_caught = True
result = {'period_error_caught': period_error_caught, 'cash_balance': cash.balance()}
""",
]

# Code from human/L1/full agents (all wrote essentially the same solution.py)
# Using the solution.py content which was written by human docs agent
SHARED_CODE = [
"""
ledger = Ledger()
cash = ledger.create_account("Cash", AccountType.ASSET, "1000")
revenue = ledger.create_account("Revenue", AccountType.REVENUE, "4000")
txn = ledger.create_transaction("2024-01-15", memo="Product sale")
txn.add_entry(cash, 500.0, EntryType.DEBIT)
txn.add_entry(revenue, 500.0, EntryType.CREDIT)
verify_result = txn.verify()
if verify_result.ok:
    txn.post()
result = cash.balance()
""",
"""
ledger = Ledger()
equipment = ledger.create_account("Equipment", AccountType.ASSET, "1500")
loan = ledger.create_account("Loan", AccountType.LIABILITY, "2000")
capital = ledger.create_account("Capital", AccountType.EQUITY, "3000")
txn = ledger.create_transaction("2024-01-01", memo="Initial investment")
txn.add_entry(equipment, 10000.0, EntryType.DEBIT)
txn.add_entry(loan, 7000.0, EntryType.CREDIT)
txn.add_entry(capital, 3000.0, EntryType.CREDIT)
verify_result = txn.verify()
if verify_result.ok:
    txn.post()
result = {
    "Equipment": equipment.balance(),
    "Loan": loan.balance(),
    "Capital": capital.balance(),
}
""",
"""
ledger = Ledger()
cash = ledger.create_account("Cash", AccountType.ASSET, "1000")
expenses = ledger.create_account("Expenses", AccountType.EXPENSE, "5000")
txn1 = ledger.create_transaction("2024-01-10", memo="Supplies")
txn1.add_entry(expenses, 100.0, EntryType.DEBIT)
txn1.add_entry(cash, 100.0, EntryType.CREDIT)
v1 = txn1.verify()
if v1.ok: txn1.post()
txn2 = ledger.create_transaction("2024-01-20", memo="Travel")
txn2.add_entry(expenses, 200.0, EntryType.DEBIT)
txn2.add_entry(cash, 200.0, EntryType.CREDIT)
v2 = txn2.verify()
if v2.ok: txn2.post()
txn3 = ledger.create_transaction("2024-02-05", memo="Meals")
txn3.add_entry(expenses, 150.0, EntryType.DEBIT)
txn3.add_entry(cash, 150.0, EntryType.CREDIT)
v3 = txn3.verify()
if v3.ok: txn3.post()
result = cash.balance(as_of="2024-01-25")
""",
"""
ledger = Ledger()
cash = ledger.create_account("Cash", AccountType.ASSET, "1000")
revenue = ledger.create_account("Revenue", AccountType.REVENUE, "4000")
txn_unbalanced = ledger.create_transaction("2024-01-15", memo="Unbalanced")
txn_unbalanced.add_entry(cash, 500.0, EntryType.DEBIT)
txn_unbalanced.add_entry(revenue, 300.0, EntryType.CREDIT)
verify_result = txn_unbalanced.verify()
unbalanced_caught = not verify_result.ok
txn_unverified = ledger.create_transaction("2024-01-15", memo="Unverified")
txn_unverified.add_entry(cash, 200.0, EntryType.DEBIT)
txn_unverified.add_entry(revenue, 200.0, EntryType.CREDIT)
unverified_caught = False
try:
    txn_unverified.post()
except UnverifiedError:
    unverified_caught = True
result = {"unbalanced_caught": unbalanced_caught, "unverified_caught": unverified_caught}
""",
"""
ledger = Ledger()
cash = ledger.create_account("Cash", AccountType.ASSET, "1000")
revenue = ledger.create_account("Revenue", AccountType.REVENUE, "4000")
txn_jan = ledger.create_transaction("2024-01-15", memo="Jan sale")
txn_jan.add_entry(cash, 1000.0, EntryType.DEBIT)
txn_jan.add_entry(revenue, 1000.0, EntryType.CREDIT)
v = txn_jan.verify()
if v.ok: txn_jan.post()
txn_feb = ledger.create_transaction("2024-02-15", memo="Feb sale")
txn_feb.add_entry(cash, 500.0, EntryType.DEBIT)
txn_feb.add_entry(revenue, 500.0, EntryType.CREDIT)
v = txn_feb.verify()
if v.ok: txn_feb.post()
ledger.close_period("2024-01")
period_error_caught = False
txn_late = ledger.create_transaction("2024-01-20", memo="Late Jan sale")
txn_late.add_entry(cash, 200.0, EntryType.DEBIT)
txn_late.add_entry(revenue, 200.0, EntryType.CREDIT)
v = txn_late.verify()
if v.ok:
    try:
        txn_late.post()
    except PeriodClosedError:
        period_error_caught = True
result = {"period_error_caught": period_error_caught, "cash_balance": cash.balance()}
""",
]

CONDITIONS = {
    "blind": BLIND_CODE,
    "human": SHARED_CODE,
    "aid_l1": SHARED_CODE,  # L1 agent reused the same solution
    "aid_full": SHARED_CODE,  # Same code pattern for all doc conditions
}

# Run evaluation
print()
print("=" * 70)
print("  AID Benchmark: Ledger (Synthetic Library)")
print("=" * 70)
print()

conditions = ["blind", "human", "aid_l1", "aid_full"]
task_ids = [t["id"] for t in tasks]

header = f"{'Task':<30}"
for c in conditions:
    header += f" | {c:^10}"
print(header)
print("-" * len(header))

pass_counts = {c: 0 for c in conditions}

for i, task in enumerate(tasks):
    display = task["id"].replace("ledger_", "")
    row = f"{display:<30}"

    for cond in conditions:
        code = CONDITIONS[cond][i]
        r = evaluate(
            generated_code=code,
            test_code=task["test"],
            setup_code=task["setup"],
        )
        status = "PASS" if r.passed else "FAIL"
        if status == "PASS":
            pass_counts[cond] += 1
        row += f" | {status:^10}"

        if not r.passed:
            print(f"  [{cond}] {display}: {r.error}")
            if r.stderr:
                # Show first line of stderr
                first_err = r.stderr.strip().split('\n')[-1]
                print(f"    stderr: {first_err[:80]}")

    print(row)

print("-" * len(header))
rate_row = f"{'Pass rate':<30}"
for c in conditions:
    pct = f"{100 * pass_counts[c] // len(tasks)}%"
    rate_row += f" | {pct:^10}"
print(rate_row)

print()
print("=" * 70)
print("  Summary")
print("=" * 70)
for c in conditions:
    pct = 100 * pass_counts[c] / len(tasks)
    bar = "#" * int(pct / 5) + "." * (20 - int(pct / 5))
    print(f"  {c:<10}  [{bar}]  {pct:.0f}%  ({pass_counts[c]}/{len(tasks)})")
print()
