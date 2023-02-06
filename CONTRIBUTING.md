## Contributing 

- With issues:
  - Use the search tool before opening a new issue.
  - Please provide source code and commit sha if you found a bug.
  - Review existing issues and provide feedback or react to them.

- With pull requests:
  - Open your pull request against `master`
  - Your pull request should have no more than two commits, if not you should squash them.
  - It should pass all tests in the available continuous integrations systems such as TravisCI.
  - You should add/modify tests to cover your proposed code changes.
  - If your pull request contains a new feature, please document it well:
    * Consider adding Go executable examples
    * Comment all new exported types if outside of the `internal` package
    * (optional) Mentione it in the README
    * Add a comment in the CHANGELOG.md explaining your feature
