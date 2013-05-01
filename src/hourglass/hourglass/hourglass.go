package main

import (
  "fmt"
  "os"
  "os/user"
  "path"
  "database/sql"
  "text/tabwriter"
  sqlite "github.com/mattn/go-sqlite3"
  "hourglass/database"
  "hourglass/commands"
  "hourglass/clock"
)

const Usage = `hourglass is a tool for time tracking.

Usage:

    %s command [arguments]

Commands:

    status      Show any running activities
    start       Start an activity
    stop        Stop an activity

Use "%s help [command]" for more information about a command.
`

func main() {
  if len(os.Args) < 2 {
    fmt.Fprintf(os.Stderr, Usage, os.Args[0], os.Args[0])
    os.Exit(1)
  }

  help := false
  commandName := os.Args[1]
  if commandName == "help" {
    if len(os.Args) != 3 {
      fmt.Fprint(os.Stderr, "The help command requires one argument.\n")
      os.Exit(1)
    }
    help = true
    commandName = os.Args[2]
  }

  var cmd commands.Command
  switch commandName {
  case "status":
    cmd = commands.StatusCommand{}
  case "start":
    cmd = commands.StartCommand{}
  case "stop":
    cmd = commands.StopCommand{}
  }

  if help {
    fmt.Fprintf(os.Stderr, cmd.Help(), os.Args[0])
    fmt.Fprintln(os.Stderr)
    os.Exit(0)
  } else {
    /* Setup database */
    sql.Register("sqlite", &sqlite.SQLiteDriver{})

    currentUser, userErr := user.Current()
    if userErr != nil {
      fmt.Fprintln(os.Stderr, userErr)
      os.Exit(1)
    }
    dbFile := path.Join(currentUser.HomeDir, ".hourglass.db")
    db := &database.DB{"sqlite", dbFile, nil}
    migrateErr := db.Migrate()
    if migrateErr != nil {
      fmt.Fprintln(os.Stderr, migrateErr)
      os.Exit(1)
    }

    c := clock.RealClock{}
    output, err := cmd.Run(c, db, os.Args[2:]...)
    switch err.(type) {
    case nil:
      writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
      fmt.Fprintln(writer, output)
      writer.Flush()
      os.Exit(0)
    case commands.SyntaxErr:
      fmt.Fprintln(os.Stderr, err)
      fmt.Fprintf(os.Stderr, cmd.Help(), os.Args[0])
      fmt.Fprintln(os.Stderr)
      os.Exit(1)
    default:
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
    }
  }
}
