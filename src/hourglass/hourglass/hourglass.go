package main

import (
  "flag"
  "fmt"
  "os"
  "os/user"
  "path"
  "database/sql"
  "text/tabwriter"
  sqlite "github.com/mattn/go-sqlite3"
  "hourglass"
)

const Usage = `hourglass is a tool for time tracking.

Usage:

	%s [global-opts] command [arguments]

Global options:
	-sql	Use SQLite backend (default)
	-csv	Use CSV backend

Commands:

	list	List activities
	start	Start an activity
	stop	Stop an activity
	edit	Edit an activity

Use "%s help [command]" for more information about a command.
`

func printUsage() {
  fmt.Fprintf(os.Stderr, Usage, os.Args[0], os.Args[0])
}

func main() {
  sqlFlag := flag.Bool("sql", false, "Use SQLite backend")
  csvFlag := flag.Bool("csv", false, "Use CSV backend")
  flag.Parse()

  if len(flag.Args()) < 1 {
    printUsage()
    os.Exit(1)
  }

  if *sqlFlag && *csvFlag {
    fmt.Fprint(os.Stderr, "Error: -sql and -csv are mutually exclusive options\n")
    printUsage()
    os.Exit(1)
  }

  help := false
  commandName := flag.Arg(0)
  if commandName == "help" {
    if len(flag.Args()) != 2 {
      fmt.Fprint(os.Stderr, "The help command requires one argument.\n")
      os.Exit(1)
    }
    help = true
    commandName = flag.Arg(1)
  }

  var cmd hourglass.Command
  switch commandName {
  case "list":
    cmd = hourglass.ListCommand{}
  case "start":
    cmd = hourglass.StartCommand{}
  case "stop":
    cmd = hourglass.StopCommand{}
  case "edit":
    cmd = hourglass.EditCommand{}
  default:
    fmt.Fprintln(os.Stderr, "Invalid command:", commandName)
    printUsage()
    os.Exit(1)
  }

  if help {
    fmt.Fprintf(os.Stderr, cmd.Help(), os.Args[0])
    fmt.Fprintln(os.Stderr)
    os.Exit(0)
  } else {
    /* Setup database */
    currentUser, userErr := user.Current()
    if userErr != nil {
      fmt.Fprintln(os.Stderr, userErr)
      os.Exit(1)
    }

    var db hourglass.Database
    if !*csvFlag {
      sql.Register("sqlite", &sqlite.SQLiteDriver{})
      dbFile := path.Join(currentUser.HomeDir, ".hourglass.db")
      db = &hourglass.Sql{"sqlite", dbFile, nil}
    } else {
      csvFile := path.Join(currentUser.HomeDir, ".hourglass.csv")

      var csvErr error
      db, csvErr = hourglass.NewCsv(csvFile)
      if csvErr != nil {
        fmt.Fprintln(os.Stderr, csvErr)
        os.Exit(1)
      }
    }

    migrateErr := db.Migrate()
    if migrateErr != nil {
      fmt.Fprintln(os.Stderr, migrateErr)
      os.Exit(1)
    }

    c := hourglass.DefaultClock{}
    output, err := cmd.Run(c, db, flag.Args()[1:]...)
    switch err.(type) {
    case nil:
      writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
      fmt.Fprintln(writer, output)
      writer.Flush()
      os.Exit(0)
    case hourglass.ErrSyntax:
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
