# gopuntes

https://github.com/user-attachments/assets/ffad79b0-3d63-4247-9ed9-c138204fd2b2

> A flexible note-browsing tool 💖🇪🇸

**gopuntes** is a terminal-based application for browsing and viewing your local notes. It provides a fast and simple TUI (Text User Interface) to navigate through your Markdown and PDF files stored in a designated directory.

Built with Go & [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

## 🧰 Features

-   **Terminal-based UI**: Browse your notes without leaving the terminal.
-   **Multi-format Support**: Natively renders Markdown (`.md`) files and opens PDF (`.pdf`) files in your system's default viewer.
-   **Simple Setup**: On first run, `gopuntes` will prompt you to set the path to your notes directory.
-   **Fuzzy Search**: Quickly filter and find the note you're looking for.
-   **Cross-Platform**: Works on macOS & Linux.

## 📦 Installation

### From Source

Ensure you have a working Go environment (Go 1.21+ is recommended).

```sh
go install github.com/gopuntes@latest
```

or

```sh
git clone https://github.com/qrxnz/gopuntes.git && \
cd gopuntes && \
go build .
```

Alternatively, if you have `just` installed, you can simply run:

```sh
just build
```

### Using Nix ❄️

-   Try it without installing:

```sh
nix run github:qrxnz/gopuntes
```

-   Installation:

Add input in your flake like:

```nix
{
 inputs = {
   nveem = {
     url = "github:qrxnz/gopuntes";
     inputs.nixpkgs.follows = "nixpkgs";
   };
 };
}
```

With the input added you can reference it directly:

```nix
{ inputs, system, ... }:
{
  # NixOS
  environment.systemPackages = [ inputs.gopuntes.packages.${pkgs.system}.default ];
  # home-manager
  home.packages = [ inputs.gopuntes.packages.${pkgs.system}.default ];
}
```

or

You can install this package imperatively with the following command:

```nix
nix profile install github:qrxnz/gopuntes
```

### From Releases

Pre-compiled binaries for various operating systems are available on the [GitHub Releases](https://github.com/qrxnz/gopuntes/releases) page. Download the appropriate archive for your system, extract it, and place the `gopuntes` binary in your `PATH`.

## 📖 Usage

Simply run the application from your terminal:

```sh
gopuntes
```

On the first launch, you will be prompted to enter the absolute path to the directory where your notes are stored. This path is saved in `~/.config/gopuntes/config.toml`.

### Keybindings

-   **Arrow Keys** (`↑`/`↓`): Navigate the list of notes.
-   **Enter**: View the selected note. (Renders Markdown inside the TUI, opens PDFs externally).
-   **`/`**: Start filtering/searching.
-   **`q` / `esc`**: Quit the note view or the filter view.
-   **`Ctrl+C`**: Exit the application.

## 👨🏻‍💻 Development

This project uses [Nix](https://nixos.org/) with flakes and [direnv](https://direnv.net/) to provide a reproducible development environment.

1.  **Clone the repository:**

    ```sh
    git clone https://github.com/qrxnz/gopuntes.git
    cd gopuntes
    ```

2.  **Activate the environment:**
    If you have Nix and direnv installed, the environment should be activated automatically when you enter the directory. If not, run:

    ```sh
    direnv allow
    ```

3.  **Available Commands:**
    This project uses `just` as a command runner. Here are the most common commands:

    -   `just run`: Build and run the application.
    -   `just build`: Build a production binary.
    -   `just lint`: Run the linter and fix issues.
    -   `just fmt`: Format the Go code.

## 🗒️ Credits

### 🎨 Inspiration

I was inspired by:

-   [charmbracelet/glow](https://github.com/charmbracelet/glow)

## 📜 License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.
