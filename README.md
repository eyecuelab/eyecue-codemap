# eyecue-codemap

- **Links in Markdown**
    - maintain accurate links to your code from your Markdown files
- **Group blocks of code together**
    - keep physically separate blocks of code logically in sync with each other

# Links in Markdown

### Goal

You have Markdown files that link to specific lines in your code. Those links won't break when code moves around, changes, or is
deleted.

You can now know if your changes broke the links in the documentation. This helps you keep the documentation up-to-date
and in sync with the code.

It's now feasible to write documentation that lives outside the code files but doesn't become stale.

This also increases people's confidence that the documentation is up-to-date and worth reading.

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

## Linking to files vs. linking to lines

There are two flavors of links:

1. Linking to an entire file (no line number):
    * Put the magic `[eyecue-codemap]` comment at the top of the file you want to link to.
    * The only lines that may precede the magic comment are the shebang line (e.g. `#!/bin/bash`) and/or blank lines.
    * You must have at least one blank line after the magic comment.
2. Linking to a specific line in a file:
    * The line with the magic comment is the line that will be linked to.
    * Except: if the magic comment is the _only_ thing on the line (with the exception of the language's comment markers), it will link to the following line.

# Group blocks of code together

### Goal

Frequently, you will have multiple physically-separate blocks of code that must be kept logically in sync with each other. These blocks may be in different files. When you change one, you'd like to be reminded to change the other(s) too.

## Example

Assume these two functions must stay in sync with each other. For this example they are in the same file/language, but they do not have to be.

```
function foo() {
    return 42;
}

function bar() {
    return 42;
}
```

First, add the magic tag:
```
// [eyecue-codemap-group]
function foo() {
    return 42;
}

function bar() {
    return 42;
}
```

Save the file, then run `codemap-update.sh`. You'll see a randomly-generated group token has been added:

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
    return 42;
}

function bar() {
    return 42;
}
```

Next, copy that line and paste it at the end of the first block of code. Change it to `end-eyecue-codemap-group`:

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
    return 42;
}
// [end-eyecue-codemap-group:UuLzD7n96cD]

function bar() {
    return 42;
}
```

Save the file, and run `codemap-update.sh`. It will exit with an error:
```
group "UuLzD7n96cD" has changes (indicated with *):
  *  example-groups.js:1 (lines 2-4)
eyecue-codemap error: edit groups as needed, then run the "ack" command
```

Assume we have the `foo` function how we want it. To tell eyecue-codemap that we are happy with **all** existing `eyecue-codemap-group` blocks, run `codemap-update.sh ack`.
You'll see file has been updated to include the current hash:

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
  return 42;
}
// [end-eyecue-codemap-group:UuLzD7n96cD:b1b3871ebde321a184024143906db92065d13572]

function bar() {
  return 42;
}
```

Now, if you run `codemap-update.sh` again, it will no longer error (since the block of code matches the hash).

You can add the second function to the group by repeating the process (using the **same** group token):

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
  return 42;
}
// [end-eyecue-codemap-group:UuLzD7n96cD:b1b3871ebde321a184024143906db92065d13572]

// [eyecue-codemap-group:UuLzD7n96cD]
function bar() {
  return 42;
}
// [end-eyecue-codemap-group:UuLzD7n96cD]
```

Save the file and run `codemap-update.sh`.

```
group "UuLzD7n96cD" has changes (indicated with *):
     example-groups.js:1 (lines 2-4)
  *  example-groups.js:7 (lines 8-10)
eyecue-codemap error: edit groups as needed, then run the "ack" command
```

This indicates that the first block is unchanged, but the second block does not have a matching hash.

Again run `codemap-update.sh ack`, and you'll see the second block now has a hash:

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
  return 42;
}
// [end-eyecue-codemap-group:UuLzD7n96cD:b1b3871ebde321a184024143906db92065d13572]

// [eyecue-codemap-group:UuLzD7n96cD]
function bar() {
  return 42;
}
// [end-eyecue-codemap-group:UuLzD7n96cD:f5395bbfbff1c8c554093b97a073a629cb05b56a]
```

This is the state you want to be in when committing the code. Running `codemap-update.sh` in this state exits successfully, and produces no output regarding
the groups.

*time passes...*

Later on, when it's time to make changes, this is what the process looks like.

First, you change the `foo` function:

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
  return "something else";
}
// [end-eyecue-codemap-group:UuLzD7n96cD:b1b3871ebde321a184024143906db92065d13572]

// [eyecue-codemap-group:UuLzD7n96cD]
function bar() {
  return 42;
}
// [end-eyecue-codemap-group:UuLzD7n96cD:f5395bbfbff1c8c554093b97a073a629cb05b56a]
```

Running `codemap-update.sh` shows:

```
group "UuLzD7n96cD" has changes (indicated with *):
  *  example-groups.js:1 (lines 2-4)
     example-groups.js:7 (lines 8-10)
eyecue-codemap error: edit groups as needed, then run the "ack" command
```

Next, modify `bar`:

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
  return "something else";
}
// [end-eyecue-codemap-group:UuLzD7n96cD:b1b3871ebde321a184024143906db92065d13572]

// [eyecue-codemap-group:UuLzD7n96cD]
function bar() {
  return "something else";
}
// [end-eyecue-codemap-group:UuLzD7n96cD:f5395bbfbff1c8c554093b97a073a629cb05b56a]
```

Then run `codemap-update.sh`:
```
group "UuLzD7n96cD" has changes (indicated with *):
  *  example-groups.js:1 (lines 2-4)
  *  example-groups.js:7 (lines 8-10)
eyecue-codemap error: edit groups as needed, then run the "ack" command
```

Finally, when you're happy with each block of code, run `codemap-update.sh ack`. You'll see the hashes have been updated:

```
// [eyecue-codemap-group:UuLzD7n96cD]
function foo() {
  return "something else";
}
// [end-eyecue-codemap-group:UuLzD7n96cD:3660076f86fcab8f118515b1caf3f3123467fe9c]

// [eyecue-codemap-group:UuLzD7n96cD]
function bar() {
  return "something else";
}
// [end-eyecue-codemap-group:UuLzD7n96cD:3be4ca7b3a0b2322b6a4ef9598f04d1430b1910c]
```

# Installation

To integrate this into your repo, adapt `codemap-update.sh` to your needs. This script contains all the code needed
to get the `eyecue-codemap` executable onto the local machine. It will check for updates daily. The developer doesn't need
to install or manage anything.

See the [Powur Vision repo](https://github.com/eyecuelab/powur-vision) for an example integration with
the existing linting and Git hooks.

# Errors

The updater will consider it an error when:

* There is a duplicate unique ID
* There is a link to a unique ID that cannot be found in the repo

# CI/CD

Building and pushing the Docker image to GCP Artifact Registry is done via GitHub Actions.

The `eyecue-codemap-ci@eyecue-ops.iam.gserviceaccount.com` GCP service account is used to authenticate to Google Cloud. Credentials for this account are stored in a GitHub respository secret named `GOOGLE_AUTH_JSON`.
