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
            treefmt2
            mdformat
            alejandra

            # Others
            just
            watchexec
          ];
        };
        packages.default = pkgs.buildGoModule {
          pname = "gopuntes";
          version = "0.1.0"; # You might want to manage this dynamically later
          src = self;
          vendorHash = "sha256-tOBEcWX6JpqoPl7+H0L8RT+nnRRSNQ3FPrl95OwEEJo=";
        };
      }
    );
}
