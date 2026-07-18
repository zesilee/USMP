/*
Package huawei contains ygot-generated Go structs for Huawei YANG modules
(R04: never hand-written).

Layout:
  - gen.conf    declarative generation manifest (consumed by make gen-yang)
  - all.gen.go  the single generated compilation unit (do not edit)
  - doc.go      this file

Supported YANG modules: huawei-vlan, huawei-ifm, huawei-system,
huawei-pub-type, huawei-extension (plus modules they import, e.g.
huawei-network-instance).

To add a module: append it to modules= in gen.conf, then run

	make gen-yang VENDOR=huawei

Regeneration reads the in-repo snd/ce6866p-yang model source
(no submodule required). CI verifies the generated
output is reproducible via regen-and-diff; see
backend/internal/generated/README.md.
*/
package huawei
