# Secure Voice Calls in CLI: Feasibility Study

## Can calls be added while staying CLI-only?

Yes. Voice calls can be added without a GUI by using terminal commands and background audio pipelines.

## Recommended Technical Direction

1. Signaling
- Reuse current server protocol for call signaling (`call_offer`, `call_answer`, `ice_candidate`, `call_end`).
- Keep signaling metadata typed and versioned.

2. Media Transport
- Use WebRTC media stack (SRTP + DTLS) for NAT traversal and secure transport.
- CLI app controls call lifecycle; media runs in process with Go libraries (for example, Pion).

3. Audio I/O in CLI
- Capture microphone via platform audio backend (PulseAudio/PipeWire/CoreAudio/ALSA wrappers).
- Play received audio via corresponding output backend.
- Keep all controls as slash commands:
  - `/call <user>`
  - `/accept`
  - `/reject`
  - `/hangup`
  - `/mute`

## Security Requirements for Calls

- End-to-end media key agreement through WebRTC DTLS-SRTP.
- Optional SAS (short authentication string) shown in terminal for out-of-band verification.
- Strict codec and packet validation to reduce parser attack surface.
- Rate-limits and call invite cooldowns to prevent call spam.

## Risks and Mitigations

- NAT/firewall failures: provide TURN fallback.
- Audio device fragmentation: add backend abstraction + capability probe command.
- CPU spikes on weak devices: expose codec bitrate controls in CLI.

## Suggested Rollout

1. Phase 1: signaling-only + mock call states (no audio)
2. Phase 2: one-to-one encrypted audio call
3. Phase 3: call reliability (reconnect, jitter buffering tuning)
4. Phase 4: optional group call prototype

## Conclusion

Secure CLI calls are feasible and practical. The cleanest path is a typed signaling extension plus WebRTC-based media plane, while preserving terminal-only UX.
