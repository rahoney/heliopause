# M13-002 stale-branch probe

This documentation-only marker prepares an independent develop-targeted PR.
Its starting base is `2e196a188adce22005fce201c23da1dc27a43426`.
It changes no product code, workflow, protection setting or release input.

Keep this feature head unchanged while a separately authorized PR advances
develop. Capture this PR's own Required/CLA results before and after that
advancement, and distinguish freshness rejection from other merge blockers.

Creating this marker or observing mergeability does not qualify stale-branch
rejection. Any merge or rejection attempt requires separate user approval.
