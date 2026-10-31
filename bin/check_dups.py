python3 -c "
import csv, sys
from collections import Counter
ids = []
empty = 0
with open(sys.argv[1], newline='', encoding='utf-8-sig') as f:
    for row in csv.DictReader(f):
        memo = (row.get('Memo') or '').split(';')[0].strip()
        if memo:
            ids.append(memo)
        else:
            empty += 1
c = Counter(ids)
dups = {k: v for k, v in c.items() if v > 1}
print(f'rows with a memo id: {len(ids)}, unique ids: {len(c)}, empty memos: {empty}')
print(f'duplicate ids: {len(dups)}')
for k, v in list(dups.items())[:20]:
    print(f'{v}x {k}')
" "$HOME/My Drive/Finances/Fidelity.csv"
