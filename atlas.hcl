env "local" {
  src = "ent://ent/schema"
  url = "postgres://portunus:portunus@localhost:5432/portunus?sslmode=disable"
  dev = "docker://postgres/15/dev?search_path=public"

  migration {
    dir = "file://ent/migrate/migrations"
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
