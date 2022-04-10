# GitHub Workflow

## Step 1: Fork in the cloud

1. Visit https://github.com/pairmesh/pairmesh
2. On the top right of the page, click the `Fork` button (top right) to create
   a cloud-based fork of the repository.

## Step 2: Clone fork to local storage

Create your clone:

Choose your development directory and properly github user name.

```sh
export dev_dir=~/devel
export user=${your-github-name}
```

```sh
mkdir -p $dev_dir
cd $dev_dir
git clone https://github.com/$user/pairmesh.git
# or: git clone git@github.com:$user/pairmesh.git

cd $dev_dir/pairmesh
git remote add upstream https://github.com/pairmesh/pairmesh.git
# or: git remote add upstream git@github.com:pairmesh/pairmesh.git

# Never push to the upstream master.
git remote set-url --push upstream no_push

# Confirm that your remotes make sense:
# It should look like:
# origin    git@github.com:$(user)/pairmesh.git (fetch)
# origin    git@github.com:$(user)/pairmesh.git (push)
# upstream  https://github.com/pairmesh/pairmesh (fetch)
# upstream  no_push (push)
git remote -v
```

## Step 3: Branch

Get your local master up to date:

```sh
cd $dev_dir/pairmesh
git fetch upstream
git checkout master
git rebase upstream/master
```

Branch from master:

```sh
git checkout -b myfeature
```

## Step 4: Develop

### Edit the code

You can now edit the code on the `myfeature` branch.

### Test

Build and run all tests:

```sh
make

make check

make test
```

## Step 5: Keep your branch in sync

```sh
# While on your myfeature branch.
git fetch upstream
git rebase upstream/master
```

Please don't use `git pull` instead of the above `fetch`/`rebase`. `git pull`
does a merge, which leaves merge commits. These make the commit history messy
and violate the principle that commits ought to be individually understandable
and useful (see below). You can also consider changing your `.git/config` file
via `git config branch.autoSetupRebase` always to change the behavior of `git pull`.

## Step 6: Commit

Commit your changes.

```sh
git commit
```

Likely you'll go back and edit/build/test further, and then `commit --amend` in a
few cycles.

## Step 7: Push

When the changes are ready to review (or you just to create an offsite backup
or your work), push your branch to your fork on `github.com`:

```sh
git push --set-upstream ${your_remote_name} myfeature
```

## Step 8: Create a pull request

1. Visit your fork at `https://github.com/$user/pairmesh`.
2. Click the `Compare & Pull Request` button next to your `myfeature` branch.
3. Fill in the required information in the PR template.

### Get a code review

If your pull request (PR) is opened, it will be assigned to one or more
reviewers. Those reviewers will do a thorough code review, looking at
correctness, bugs, opportunities for improvement, documentation and comments,
and style.

To address review comments, you should commit the changes to the same branch of
the PR on your fork

### Revert a commit

In case you wish to revert a commit, follow the instructions below:

> NOTE: If you have upstream write access, please refrain from using the Revert
> button in the GitHub UI for creating the PR, because GitHub will create the
> PR branch inside the main repository rather than inside your fork.

Create a branch and synchronize it with the upstream:

```sh
# create a branch
git checkout -b myrevert

# sync the branch with upstream
git fetch upstream
git rebase upstream/master

# SHA is the hash of the commit you wish to revert
git revert SHA
```

This creates a new commit reverting the change. Push this new commit to
your remote:

```sh
git push ${your_remote_name} myrevert
```

Create a PR based on this branch.

### Cherry pick a commit to a release branch

In case you wish to cherry pick a commit to a release branch, follow the
instructions below:

Create a branch and synchronize it with the upstream:

```sh
# sync the branch with upstream.
git fetch upstream

# checkout the release branch.
# ${release_branch_name} is the release branch you wish to cherry pick to.
git checkout upstream/${release_branch_name}
git checkout -b my-cherry-pick

# cherry pick the commit to my-cherry-pick branch.
# ${SHA} is the hash of the commit you wish to revert.
git cherry-pick ${SHA}

# push this branch to your repo, file an PR based on this branch.
git push --set-upstream ${your_remote_name} my-cherry-pick
```
