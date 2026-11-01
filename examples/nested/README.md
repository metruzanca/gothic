# Nested stateful components

A stateful `Dashboard` component contains three stateful `Counter` children:

- one static child,
- one child inside an `if` branch,
- keyed children in a `for` list.

Each counter keeps its own state. The branch child resets when the branch
closes. List children keep their state while items update. A counter emits its
message straight to the App.

Run it:

```sh
go run .
```

Then open <http://localhost:8080>.

What to try:

1. Increment the static counter, alpha, and beta.
2. Add a list child and increment it. Existing children keep their state.
3. Toggle the branch child, increment it, then toggle it off and on. Its state
   resets because the branch unmounts.
4. Pick a counter. The App shows the picked label, even though the message came
   from a nested child.
5. Toggle the dashboard. The whole nested tree unmounts and resets.
