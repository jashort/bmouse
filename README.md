# BMouse

A command-line tool for controlling the RGB lighting and scroll wheel mode on Razer Basilisk V3 mice via direct USB HID — no Razer drivers or Synapse required.

[Demo](https://github.com/user-attachments/assets/8502f784-ff05-4f9f-9947-c9e7cd181305)

## Supported Devices

Tested with: Razer Basilisk V3 Pro (Wireless), plugged in via USB.


## Alternatives:
- [OpenRazer](https://openrazer.github.io/) — Linux driver and user-space daemon for Razer devices, with Python bindings and a command-line tool (`razer-cli`) for controlling lighting and other features. Linux only.

## Installation

```sh
go install github.com/jashort/bmouse@latest
```

Or build from source:

```sh
git clone https://github.com/jashort/bmouse.git
cd bmouse
make build
```

> **macOS** You may need to grant your terminal `Input Monitoring` permission in System Preferences > Security & Privacy > Privacy to allow `bmouse` to detect the mouse.
>
> For example:
> ```sh
> ./bmouse list
> Razer HID devices: PID=0x00AA  Razer Basilisk V3 Pro           UsagePage=0x0001  Usage=0x0006  Interface=2  Path=DevSrvsID:4294971771
> ...
> ./bmouse static ff0000
> no Razer Basilisk V3 found (is it plugged in?)
> ```
> Indicates that the mouse is detected but not accessible due to missing permissions. Granting `Input Monitoring` permission should resolve this.

## Usage

```
bmouse <command> [--zone <zone>] [args...]
```

### Commands

| Command                              | Description                                | Example                             |
|--------------------------------------|--------------------------------------------|-------------------------------------|
| `list`                               | List all Razer HID devices                 | `bmouse list`                       |
| `status`                               | Show current active-profile settings  | `bmouse status`                     |
| `static <hex-color>`                 | Set a solid color                          | `bmouse static ff0000`              |
| `breathe <hex-color>`                | Single-color breathing                     | `bmouse breathe 00ff00`             |
| `breathe-dual <color1> <color2>`     | Two-color breathing                        | `bmouse breathe-dual ff0000 0000ff` |
| `spectrum`                           | Rainbow spectrum cycling                   | `bmouse spectrum`                   |
| `reactive <hex-color> [--speed 1-3]` | Light up on click                          | `bmouse reactive ff0000 --speed 1`  |
| `off`                                | Turn LEDs off                              | `bmouse off`                        |
| `brightness [0-255]`                 | Get or set brightness                      | `bmouse brightness 200`             |
| `scroll [mode]`                      | Get or set scroll wheel mode               | `bmouse scroll tactile`             |

#### Reactive speed values

| Value | Duration         |
|-------|------------------|
| `1`   | Short            |
| `2`   | Medium (default) |
| `3`   | Long             |

#### Scroll wheel modes

| Mode      | Description                                               |
|-----------|-----------------------------------------------------------|
| `tactile` | Clicky, notched scrolling                                 |
| `free`    | Free-spin (smooth, silent)                                |
| `smart`   | Smart Reel — automatically switches based on scroll speed |

### Flags

| Flag              | Description                                                        | Default        |
|-------------------|--------------------------------------------------------------------|----------------|
| `--zone <zone>`   | Target LED zone (see table below)                                  | `all`          |
| `--speed <1-3>`   | Reactive effect duration (1=short, 2=medium, 3=long)              | `2` (medium)   |

### Zones

Use the optional `--zone` flag to target a specific LED zone. Defaults to all zones.

| Zone     | Description                |
|----------|----------------------------|
| `all`    | All LEDs at once (default) |
| `scroll` | Scroll-wheel LED           |
| `logo`   | Logo LED                   |
| `under`  | Underglow light strip      |

### Color format

6-digit hex, with or without a leading `#`:

```
ff8800    #ff8800
```

## Examples

```sh
# Set all LEDs to red
bmouse static ff0000

# Green breathing effect on the logo only
bmouse breathe --zone logo 00ff88

# Two-color breathing
bmouse breathe-dual ff0000 0000ff

# Red reactive with short duration
bmouse reactive ff0000 --speed 1

# Spectrum cycling on the scroll wheel
bmouse spectrum --zone scroll

# Turn off underglow
bmouse off --zone under

# Set brightness to 200
bmouse brightness 200

# Check current brightness
bmouse brightness

# Switch to free-spin scroll wheel
bmouse scroll free


# Check current scroll mode
bmouse scroll

# Show current LED and scroll settings
bmouse status
```

## Dependencies

- [go-hid](https://github.com/sstallion/go-hid) — cross-platform Go bindings for the HIDAPI library

## License

MIT

