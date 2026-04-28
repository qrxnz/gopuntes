{
  inputs.utils.url = "github:numtide/flake-utils";

  outputs = {
    self,
    nixpkgs,
    utils,
  }:
    utils.lib.eachDefaultSystem (
      system: let
        pkgs = import nixpkgs {inherit system;};
      in {
        devShells.default = pkgs.mkShell rec {
          buildInputs = with pkgs; [
            # Go
            go
            gopls
            delve

            # Formatters
            treefmt
            mdformat
            alejandra
            prettier

            # Others
            go-task
          ];
        };
        packages.default = pkgs.buildGoModule {
          pname = "gopuntes";
          version = "1.0.5";
          src = self;
          vendorHash = "sha256-LkcZ/WwNHKC4fxf0OShE+x+qDmh2vYIJ1x8QQJKM44A=";
        };
      }
    );
}
