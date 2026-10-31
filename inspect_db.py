import sqlite3
con = sqlite3.connect('mfs100.db')
print("Tables:", con.execute("SELECT name FROM sqlite_master WHERE type='table'").fetchall())
for table in ['users', 'credentials', 'windows_credentials']:
    try:
        print(f"\n--- {table} ---")
        rows = con.execute(f"SELECT * FROM {table}").fetchall()
        print(f"Row count: {len(rows)}")
        for r in rows:
            print(r)
    except Exception as e:
        print(f"Error reading {table}: {e}")
