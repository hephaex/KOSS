# Why `gitsync` is a thing

Fuchsia source is mirrored to GitHub, and we use Gerrit for code reviews.
Rather than run our own Gerrit server, we let Google do that for us.  However,
Google's Gerrit infrastructure is designed to run on Git-on-Borg and doesn't
have the ability to push to GitHub directly (yet; see b/10295007.)  So we
run `gitsync` on a cron to sync between our Git-on-Borg host and GitHub.

Fuchsia's Git-on-Borg host is public: https://fuchsia.googlesource.com/

Fuchsia is mirrored to GitHub at https://github.com/fuchsia-mirror

Git-on-Borg has some other features that are helpful to Fuchsia, like the
ability to slam it with automated build and test bots without fear of angering
the nice folks at GitHub.

If b/10295007 is ever resolved and GoB/Gerrit can push to GitHub, we can take
off and nuke the entire script from orbit (it's the only way to be sure.)
