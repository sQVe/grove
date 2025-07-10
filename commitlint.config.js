export default {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Ensure subject case is lower case
    'subject-case': [2, 'always', 'lower-case'],
    // Ensure subject is not empty
    'subject-empty': [2, 'never'],
    // Ensure subject is not too long
    'subject-max-length': [2, 'always', 72],
    // Ensure type is not empty
    'type-empty': [2, 'never'],
    // Allowed commit types
    'type-enum': [
      2,
      'always',
      [
        'feat',     // New feature
        'fix',      // Bug fix
        'docs',     // Documentation changes
        'style',    // Code style changes (formatting, etc.)
        'refactor', // Code refactoring
        'perf',     // Performance improvements
        'test',     // Adding or updating tests
        'chore',    // Maintenance tasks
        'ci',       // CI/CD changes
        'build',    // Build system changes
        'revert'    // Revert previous commit
      ]
    ]
  }
};