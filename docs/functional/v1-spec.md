# Spotpilot v1 Functional Plan

## 1. Goal

Spotpilot is a local CLI tool for agents to control Spotify playback on a user’s machine.

Primary intended flow:

* User installs Spotpilot via Homebrew
* User talks to their agent in natural language
* The agent translates that into Spotpilot CLI calls such as:

  * `spotpilot play "Master of Puppets"`
  * `spotpilot pause`
  * `spotpilot status`

Spotpilot is **not** the conversational interface. The agent is.
Spotpilot is the deterministic execution tool behind the agent.

---

## 2. Product Positioning

### v1 target

* Agent-first CLI
* Reusable by other users
* macOS and Linux first
* Homebrew-installable
* Local-machine playback only

### v1 guiding principle

Keep the first version narrow, reliable, and deterministic for agent usage.

---

## 3. Core User Experience

### First-time setup

A user installs Spotpilot and either:

* explicitly runs:

  * `spotpilot login`

or

* directly runs:

  * `spotpilot play "Master of Puppets"`

If no valid session exists, Spotpilot automatically triggers the login flow.

### Login flow

* Spotpilot checks whether a valid local saved session already exists
* If yes, login is skipped and success is returned
* If no, Spotpilot launches a supported browser:

  1. Google Chrome
  2. Chromium
* Spotpilot opens Spotify login in that browser
* User logs into Spotify
* Spotpilot imports the local browser session/cookies
* Spotpilot stores the session locally
* If login was triggered from `play` or another command, Spotpilot automatically resumes the original command

### Playback flow

Example:

* User says to the agent:

  * “Please play Master of Puppets on Spotify”
* Agent runs:

  * `spotpilot play "Master of Puppets"`
* Spotpilot:

  * ensures login/session exists
  * searches Spotify
  * resolves result type using track -> album -> artist priority
  * launches Spotify desktop app if needed
  * waits briefly for local Spotify device
  * falls back to Spotify web player if desktop device never appears
  * plays the selected result on the local machine’s device
  * returns structured JSON

---

## 4. Scope of v1 Commands

Spotpilot v1 supports:

* `spotpilot login`
* `spotpilot play "<query>"`
* `spotpilot pause`
* `spotpilot resume`
* `spotpilot next`
* `spotpilot previous`
* `spotpilot status`
* `spotpilot version`

Not included in v1:

* `logout`
* playlists as first-class commands
* queue management
* library-aware ranking
* device selection
* remote device takeover
* browser support beyond Chrome/Chromium for session import
* interactive disambiguation
* natural language parsing inside Spotpilot

---

## 5. Command Behavior

### 5.1 `spotpilot login`

#### Behavior

* Checks for a valid saved local session
* If valid session exists:

  * returns success
  * does not reopen login
* If no valid session exists:

  * launches Google Chrome if installed
  * otherwise launches Chromium if installed
  * otherwise fails
  * opens Spotify login
  * waits for user authentication with a bounded timeout
  * imports local browser session/cookies
  * saves session locally

#### Browser policy

* Supported for session import in v1:

  * Google Chrome
  * Chromium
* If neither Chrome nor Chromium is installed:

  * fail clearly
* Default browser is **not** used for session import in v1

---

### 5.2 `spotpilot play "<query>"`

#### Input policy

* The query is treated as a **raw query string**
* Spotpilot does **not** clean or reinterpret messy natural language in v1
* Query cleanup is the responsibility of the agent

Example:

* expected:

  * `spotpilot play "Master of Puppets"`
* not expected:

  * Spotpilot interpreting `"please play master of puppets on spotify"` intelligently

#### Search policy

Searches Spotify globally only.

No use of:

* liked songs
* saved albums
* personal library
* historical listening preferences

#### Matching policy

Result resolution order is:

1. track
2. album
3. artist
4. not found

If multiple matches exist within a category:

* Spotpilot selects the **top Spotify result automatically**

No disambiguation or user-choice prompts in v1.

#### Playback policy

* One-shot playback only
* Selected result is played immediately
* Current local playback may be replaced directly
* No queue insertion behavior

#### Device policy

* Spotpilot always prefers the **local machine’s Spotify device**
* Spotpilot must not hijack remote devices such as:

  * phones
  * speakers
  * other computers

#### Recovery flow

If a local playable device is not immediately available:

1. Try local Spotify desktop app first
2. Wait for local device to appear with a bounded timeout
3. If no local desktop device appears:

   * open Spotify web player
4. Wait for browser player device with a bounded timeout
5. If still unavailable:

   * fail

#### Login auto-trigger

If no valid session exists:

* `play` internally triggers the login flow
* after successful login, it automatically continues the original play request

#### Failure behavior

For v1, the user-facing failure behavior is intentionally simple:

* if no track/album/artist match is found:

  * return not found
* other internal failures should not expose unnecessary detail to the user-facing surface

---

### 5.3 `spotpilot pause`

#### Behavior

* Same login/session handling as `play`
* Same recovery behavior as `play`
* Prefers local desktop app, then web player fallback
* Acts only on the local machine’s Spotify device

---

### 5.4 `spotpilot resume`

#### Behavior

* Same login/session handling as `play`
* Same recovery behavior as `play`
* Prefers local desktop app, then web player fallback
* Acts only on the local machine’s Spotify device

---

### 5.5 `spotpilot next`

#### Behavior

* Same login/session handling as `play`
* Same recovery behavior as `play`
* Prefers local desktop app, then web player fallback
* Acts only on the local machine’s Spotify device

---

### 5.6 `spotpilot previous`

#### Behavior

* Same login/session handling as `play`
* Same recovery behavior as `play`
* Prefers local desktop app, then web player fallback
* Acts only on the local machine’s Spotify device

---

### 5.7 `spotpilot status`

#### Behavior

`status` is side-effect free.

It should:

* report login state
* report active device
* report current track if applicable

It should **not**:

* trigger login
* launch browser
* launch Spotify
* attempt recovery

#### Status states

Valid outcomes include:

* logged in + playing
* logged in + paused
* logged in + idle
* not logged in

#### Idle handling

If Spotify is open but nothing is currently playing:

* return `idle`
* do not treat as an error

#### Minimal status payload contents

* logged-in state
* active device
* current track

No extra backend or storage metadata in v1.

---

### 5.8 `spotpilot version`

#### Behavior

Returns the Spotpilot version.

Included in v1:

* `spotpilot version`

Not required in v1:

* `spotpilot --version`

---

## 6. Output Design

### 6.1 Default output mode

Spotpilot is **JSON-first**.

JSON is the default output because Spotpilot is primarily designed for agentic usage.

#### Rationale

* stable machine contract
* deterministic parsing
* easier tool integration
* avoids agents needing to remember `--json`

---

### 6.2 Optional human mode

Spotpilot also supports:

* `--plain`

This produces a **single concise line** of human-readable output.

#### Example

JSON default:

```
{
  "ok": true,
  "command": "play",
  "state": "playing",
  "message": "Playing Master of Puppets",
  "result": {
    "match_type": "track",
    "title": "Master of Puppets",
    "artist": "Metallica"
  }
}
```

Plain mode:

```
Playing: Master of Puppets — Metallica
```

---

### 6.3 JSON consistency

Every command in v1 returns the same top-level JSON structure.

#### Required top-level fields

* `ok`
* `command`
* `state`
* `message`

#### Result consistency

Where applicable, commands should reuse the same result shape for consistency.

`status` should align with the general response model rather than inventing a separate incompatible payload.

#### Minimal payload principle

Keep JSON payloads minimal in v1.
Do **not** include extra Spotify URLs/URIs unless needed later.

---

## 7. Exit Codes

Failures should use:

* JSON with `ok: false`
* non-zero process exit code

This gives agents two reliable signals:

* structured payload
* process status

---

## 8. Session and Storage

### 8.1 Authentication/session model

Spotpilot v1 uses a **browser-session / cookie import approach** rather than Spotify Developer OAuth as the primary path.

#### Chosen login UX

* open Spotify login in Chrome/Chromium
* user signs in
* Spotpilot imports session/cookies
* session is stored locally
* future commands reuse saved local session

---

### 8.2 Storage model

Use a **hybrid storage approach**:

1. OS keychain/keyring if available
2. protected local file fallback if secure store is unavailable

#### Rationale

This balances:

* better security where available
* practical cross-platform behavior on macOS/Linux
* simpler v1 operability

---

## 9. Browser Rules

### Supported for login/session import in v1

* Google Chrome
* Chromium

### Preference order

1. Google Chrome
2. Chromium

### Unsupported in v1 for cookie/session import

* Firefox
* Safari
* arbitrary default browser

### Default browser usage

Default browser is acceptable for **web player playback fallback**, but not for session import.

---

## 10. Playback Device Rules

### Device targeting

* Always target the local machine’s Spotify device
* Never intentionally hijack remote devices in v1

### Local recovery order

1. Existing local Spotify device if already available
2. Launch local Spotify desktop app
3. Wait briefly for desktop device
4. Open Spotify web player
5. Wait briefly for browser device
6. Fail if no local device appears

---

## 11. Timeout Rules

All waiting should be bounded in v1.

### Required bounded waits

1. login completion timeout
2. desktop-app device appearance timeout
3. web-player device appearance timeout

Spotpilot must not hang indefinitely.

---

## 12. Error / State Simplification

### User-facing simplicity

v1 should keep user-facing outcomes simple.

Key user-facing states include:

* playing
* paused
* idle
* not_found
* not_logged_in
* error

### Not found behavior

If no matching track, album, or artist exists:

* return not found

### Internal detail minimization

Avoid exposing too much backend detail in v1 responses.

---

## 13. Agent Integration Contract

Spotpilot is designed to be called by an agent as a local tool.

### Intended usage

User says:

* “Play Master of Puppets on Spotify”

Agent executes:

* `spotpilot play "Master of Puppets"`

Spotpilot returns structured JSON.
The agent then decides what to say back to the user.

### Responsibility split

**Agent responsibilities**

* understand natural language
* clean/rewrite user phrasing if needed
* decide which Spotpilot command to run
* present final response to the user

**Spotpilot responsibilities**

* login/session management
* Spotify session import
* search
* match resolution
* local playback control
* deterministic structured output

---

## 14. Non-Goals for v1

These are explicitly out of scope:

* full public developer-platform architecture
* logout flow
* multi-browser cookie/session import
* remote device control
* user library ranking
* queue management
* playlists as a dedicated playback target command
* interactive selection among results
* natural language parsing inside Spotpilot
* storage/backend diagnostics in `status`
* detailed failure taxonomy exposed to end users
* `--version`
* dedicated `help` command

---

## 15. Summary of Final v1 Decisions

### Platform

* macOS + Linux first

### Distribution

* Homebrew-installable CLI

### Main interface

* agent calls Spotpilot as a CLI tool

### Commands

* `login`
* `play`
* `pause`
* `resume`
* `next`
* `previous`
* `status`
* `version`

### Playback resolution

* track -> album -> artist -> not found

### Match selection

* top Spotify result automatically

### Search scope

* global Spotify only

### Output

* JSON by default
* `--plain` optional
* single concise line in plain mode

### JSON fields

* `ok`
* `command`
* `state`
* `message`

### Status payload

* login state
* active device
* current track

### Idle

* valid state, not an error

### Device preference

* local machine only

### Recovery order

* desktop app first
* web player second

### Login/browser policy

* Chrome first
* Chromium second
* fail if neither installed
* session import only from Chrome/Chromium

### Session persistence

* hybrid storage:

  * keychain/keyring first
  * protected file fallback

### Auto-login

* `play`, `pause`, `resume`, `next`, `previous` auto-trigger login if needed

### `status`

* does not trigger login
* reports not logged in if no valid session exists

### Timeouts

* bounded login timeout
* bounded desktop device timeout
* bounded web-player timeout

### Failure signaling

* JSON with `ok: false`
* non-zero exit code

### Playback semantics

* one-shot playback
* directly replace current local playback

---

## 16. Canonical Example Flows

### Example A: First-time play

```
Agent runs: spotpilot play "Master of Puppets"
-> no valid session
-> Spotpilot launches Chrome
-> user logs into Spotify
-> session imported and stored
-> Spotpilot resumes play flow
-> launches Spotify desktop if needed
-> finds local device
-> plays top matching track
-> returns JSON success
```

### Example B: Already logged in

```
Agent runs: spotpilot play "Ride the Lightning"
-> saved valid session exists
-> no login needed
-> Spotpilot resolves top track/album/artist by priority
-> plays on local device
-> returns JSON success
```

### Example C: Status while idle

```
Agent runs: spotpilot status
-> logged in
-> Spotify open or available
-> nothing currently playing
-> returns state: idle
```

### Example D: Not logged in status

```
Agent runs: spotpilot status
-> no valid session
-> returns not logged in
-> does not open browser
```

### Example E: Not found

```
Agent runs: spotpilot play "some nonexistent thing"
-> no track match
-> no album match
-> no artist match
-> returns not_found
```
