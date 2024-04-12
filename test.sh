# Check if the branch exists
if git show-ref --verify --quiet refs/remotes/origin/reuse-metadata-proposal; then
  # check if the branch has changes to the main branch
  if ! git diff --quiet origin/main..origin/reuse-metadata-proposal; then
    echo "Changes detected, creating PR..."
  else
    echo "No changes in branch, skipping PR creation"
  fi
else
  echo "Branch does not exist, skipping PR creation"
fi