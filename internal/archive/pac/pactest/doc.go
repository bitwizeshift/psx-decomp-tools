/*
Package pactest provides helpers for testing code that consumes the pac package.

It assembles PAC archives in memory from nested [Node] specifications built with
[Leaf], [Dir], and [Raw], renders them with [Build], and compares parsed payloads
by their identifying fields with [CompareFiles].
*/
package pactest
