# radcliffe
---

#### JSON is like a box of chocolates..

`radcliffe` is a library for introspecting a JSON payload and returning metadata about that JSON payload, specifically information about its type according to the [swagger.io](http://swagger.io/specification/) specification primitives.

`radcliffe` is able to determine number formats in addition to some string and date formats. For a full list of implemented types please see [Implemented Types](#implemented-types)

### Implemented Types ###
| type | format |
| --- | --- |
| integer | int32 |
| integer | int64 |
| number | float |
| number | double |
| boolean | |
| string |
| string | date |
| string | date-time |
