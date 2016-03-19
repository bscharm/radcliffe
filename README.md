# radcliffe
---

#### Once typecasted, always typecasted

`radcliffe` is a service for introspecting a JSON payload and returning metadata about that JSON payload, specifically information about its type according to the [swagger.io](http://swagger.io/specification/) specification primitives.

`radcliffe` is able to determine number formats in addition to some string and date formats. For a full list of implemented types please see [Implemented Types](#implemented-types)

### Implemented Types ###
| type | format |
| --- | --- |
| integer | int32 |
| integer | int64 |
| number | float |
| number | double |
| boolean | |
| string | __TODO__ |
| date | __TODO__ |
| date_time | __TODO__ |
