# Rosetta Dispatcher Server

## Description

The Rosetta Dispatcher Server is a simple proxy/dispatcher which forwards the HTTP request
to appropriate spork based on block height in a  provided spork list

Current implementation of Rosetta API require block height (called Index in Rosetta nomenclature) to be provided
and this allows proxy to simply check height boundaries. If this changes, as Rosetta API permits using only block hash
a new implementation of routing mechanism should be created