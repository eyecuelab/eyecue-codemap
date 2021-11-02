# eyecue-codemap

Developer puts a special string anywhere, in any file in the repo. Use whatever language's comment syntax.

```
function foo() {
    let x = 42; // [eyecue-codemap]
```

Run an "updater" program that scans the files and adds a unique ID:

```
function foo() {
    let x = 42; // [eyecue-codemap:4vov64BcsXn]
```

In your usual README.md or other Markdown files, put that code into an HTML comment inside your link. Leave the URL
portion blank.

```
Here's the [secret<!--eyecue-codemap:4vov64BcsXn-->]() sauce.
```

Run the updater program again, and it will update the clickable link.

```
Here's the [secret<!--eyecue-codemap:4vov64BcsXn-->](example.js#L2) sauce.
```

Looks like this:

Here's the [secret<!--eyecue-codemap:4vov64BcsXn-->](example.js#L2) sauce.

# errors

The updater will consider it an error when:

* There is a duplicate unique ID
* There is a link to a unique ID that cannot be found in the repo

# end result

You have Markdown files that link to parts of the code. Those links won't break when code moves around, changes, or is
deleted.

You now know if your code changes broke the documentation. This helps you keep the documentation up-to-date and in sync
with the code.

It's now feasible to write documentation that lives outside the code files but doesn't become stale.

# details

To integrate this into your repo, adapt the `codemap-update.sh` to your needs. This script contains all the code needed
to get the eyecue-codemap executable onto the local machine. It will check for updates daily. The developer doesn't need
to install or manage anything.

See the [Powur Vision Gateway repo](https://github.com/eyecuelab/powur-vision-gateway) for an example integration with
the existing linting and Git hooks.
