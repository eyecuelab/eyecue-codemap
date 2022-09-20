# eyecue-codemap

## Goal

You have Markdown files that link to specific lines in your code. Those links won't break when code moves around, changes, or is
deleted.

You can now know if your changes broke the links in the documentation. This helps you keep the documentation up-to-date
and in sync with the code.

It's now feasible to write documentation that lives outside the code files but doesn't become stale.

This also increases people's confidence that the documentation is up-to-date and worth reading.

# Details

To integrate this into your repo, adapt `codemap-update.sh` to your needs. This script contains all the code needed
to get the `eyecue-codemap` executable onto the local machine. It will check for updates daily. The developer doesn't need
to install or manage anything.

See the [Powur Vision repo](https://github.com/eyecuelab/powur-vision) for an example integration with
the existing linting and Git hooks.

## Example

Here's how it works:

You put the magic string `[eyecue-codemap]` anywhere, in any file in the repo. Use whatever language's comment syntax to ensure
it doesn't break anything.

```
function foo() {
    let x = 42; // [eyecue-codemap]
```

Save the file, and then run `codemap-update.sh`. It notices your newly-added magic string and adds a unique ID. When you go back
to your editor, you'll see this (your random ID will be different).

```
function foo() {
    let x = 42; // [eyecue-codemap:4vov64BcsXn]
```

Next, in your usual README.md (or any other Markdown file), put that code into an HTML comment inside your link. Leave the URL
portion blank.

```
Here's the [secret<!--eyecue-codemap:4vov64BcsXn-->]() sauce.
```

Save the file, then run `codemap-update.sh` again. It will update the clickable link in your Markdown.

```
Here's the [secret<!--eyecue-codemap:4vov64BcsXn-->](example.js#L2) sauce.
```

From now on, after you make changes in your repo, just run `codemap-update.sh` to update all of the links in the Markdown.

### Example output

Here is how the link looks and behaves in the rendered Markdown:

Here's the [secret<!--eyecue-codemap:4vov64BcsXn-->](example.js#L2) sauce.

# Errors

The updater will consider it an error when:

* There is a duplicate unique ID
* There is a link to a unique ID that cannot be found in the repo

# CI/CD

Building and pushing the Docker image to GCP Artifact Registry is done via GitHub Actions.

The `eyecue-codemap-ci@eyecue-ops.iam.gserviceaccount.com` GCP service account is used to authenticate to Google Cloud. Credentials for this account are stored in a GitHub respository secret named `GOOGLE_AUTH_JSON`.
