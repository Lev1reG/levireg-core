# Username is the User primary key

`auth.users` is keyed by `username` (a natural key) rather than a surrogate id, and other tables reference Users by username. We chose this for readable rows and joins-free lookups in a small personal service. The cost is that a username can never change: every referencing table stores it, so renaming would mean cascading updates everywhere. Treat usernames as immutable; if renaming is ever required, migrate to a surrogate key first.
