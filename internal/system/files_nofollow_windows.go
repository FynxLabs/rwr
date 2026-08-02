//go:build windows

package system

// Windows has no O_NOFOLLOW; symlink creation there needs a privilege ordinary
// users do not hold, so the planted-symlink redirect this guards against is not
// available to an unprivileged local user in the first place.
const noFollow = 0
