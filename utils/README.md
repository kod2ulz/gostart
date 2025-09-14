# Utilities (`utils`)

This package provides a collection of miscellaneous helper functions and utilities that are used across the `gostart` library. The goal of this package is to house small, self-contained, and widely applicable tools that do not belong to a more specific domain package like `config` or `collections`.

Over time, components may be moved out of `utils` into new or existing packages if they grow in complexity and scope.

## Sub-modules

The package includes utilities for handling:

- **Errors (`error.go`):** Functions for wrapping and inspecting errors.
- **JSON (`json.go`):** Helpers for marshaling and unmarshaling JSON, including handling `null` values.
- **Networking (`net.go`):** Network-related utilities.
- **Null Types (`null.go`):** Helpers for working with nullable types, often for database interactions.
- **Numerics (`numeric.go`):** Functions for working with numbers.
- **Strings (`strings.go`):** A variety of string manipulation and generation functions.
- **Structs (`struct.go`):** Utilities for working with Go structs, such as copying data.
- **Tasks (`tasks.go`):** Tools for managing background tasks and goroutines.
- **Testing (`test.go`):** Helpers to simplify writing tests.
- **Time (`time.go`, `timer.go`):** Utilities for working with time and timers.
- **UUIDs (`uuid.go`):** Functions for generating and working with UUIDs.
- **Validation (`validator.go`):** A utility for validating the structure of objects.
