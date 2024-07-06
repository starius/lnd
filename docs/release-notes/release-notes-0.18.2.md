# Release Notes
- [Bug Fixes](#bug-fixes)
- [New Features](#new-features)
    - [Functional Enhancements](#functional-enhancements)
    - [RPC Additions](#rpc-additions)
    - [lncli Additions](#lncli-additions)
- [Improvements](#improvements)
    - [Functional Updates](#functional-updates)
    - [RPC Updates](#rpc-updates)
    - [lncli Updates](#lncli-updates)
    - [Breaking Changes](#breaking-changes)
    - [Performance Improvements](#performance-improvements)
- [Technical and Architectural Updates](#technical-and-architectural-updates)
    - [BOLT Spec Updates](#bolt-spec-updates)
    - [Testing](#testing)
    - [Database](#database)
    - [Code Health](#code-health)
    - [Tooling and Documentation](#tooling-and-documentation)

# Bug Fixes

* `closedchannels` now [successfully reports](https://github.com/lightningnetwork/lnd/pull/8800)
  settled balances even if the delivery address is set to an address that
  LND does not control.

* [SendPaymentV2](https://github.com/lightningnetwork/lnd/pull/8734) now cancels
  the background payment loop if the user cancels the stream context.

* [Fixed a bug](https://github.com/lightningnetwork/lnd/pull/8822) that caused
  LND to read the config only partially and continued with the startup.

# New Features
## Functional Enhancements
## RPC Additions

* The [SendPaymentRequest](https://github.com/lightningnetwork/lnd/pull/8734) 
  message receives a new flag `cancelable` which indicates if the payment loop 
  is cancelable. The cancellation can either occur manually by cancelling the 
  send payment stream context, or automatically at the end of the timeout period 
  if the user provided `timeout_seconds`.

## lncli Additions

* [Added](https://github.com/lightningnetwork/lnd/pull/8491) the `cltv_expiry`
  argument to `addinvoice` and `addholdinvoice`, allowing users to set the
  `min_final_cltv_expiry_delta`.

* The [`lncli wallet estimatefeerate`](https://github.com/lightningnetwork/lnd/pull/8730)
  command returns the fee rate estimate for on-chain transactions in sat/kw and
  sat/vb to achieve a given confirmation target.

# Improvements
## Functional Updates

* lnd gets new flag [`--channelbackuponshutdown`][close]. Pass the option to
  make LND update channel.backup at shutdown time in addition to regular
  updates, e.g. upon channel openings.

* The SCB file now [contains more data][close] that enable a last resort rescue
  for certain cases where the peer is no longer around.

## RPC Updates

* [`xImportMissionControl`](https://github.com/lightningnetwork/lnd/pull/8779) 
  now accepts `0` failure amounts.

* [`ChanInfoRequest`](https://github.com/lightningnetwork/lnd/pull/8813)
  adds support for channel points.

## lncli Updates

* [`importmc`](https://github.com/lightningnetwork/lnd/pull/8779) now accepts
  `0` failure amounts.

* [`getchaninfo`](https://github.com/lightningnetwork/lnd/pull/8813) now accepts
  a channel outpoint besides a channel id.

* [Fixed](https://github.com/lightningnetwork/lnd/pull/8823) how we parse the
  `--amp` flag when sending a payment specifying the payment request.

## Code Health
## Breaking Changes
## Performance Improvements

* Mission Control Store [improved performance during DB
  flushing](https://github.com/lightningnetwork/lnd/pull/8549) stage.

# Technical and Architectural Updates
## BOLT Spec Updates

* Start assuming that all hops used during path-finding and route construction
  [support the TLV onion 
  format](https://github.com/lightningnetwork/lnd/pull/8791).

* Allow channel fundee to send a [minimum confirmation depth of
  0](https://github.com/lightningnetwork/lnd/pull/8796) for a non-zero-conf
  channel. We will still wait for the channel to have at least one confirmation
  and so the main change here is that we don't error out for such a case.

## Testing
## Database

* [Fixed](https://github.com/lightningnetwork/lnd/pull/8854) pagination issues 
  in SQL invoicedb queries.

## Code Health
## Tooling and Documentation

# Contributors (Alphabetical Order)

* Andras Banki-Horvath
* Boris Nagaev
* Bufo
* Elle Mouton
* Matheus Degiovani
* Oliver Gugger
* Slyghtning

[close]: https://github.com/lightningnetwork/lnd/pull/8183
