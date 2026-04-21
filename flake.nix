{
  description = "Updo - Website monitoring tool for tracking uptime and performance";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.buildGoModule rec {
            pname = "updo";
            version = "0.4.7";

            src = ./.;

            vendorHash = "sha256-I5Cu0cXNsPoVBgouE+hRn/s1x2IbRt+V6kHDcfiRIfA=";

            # Exclude the lambda directory as it's a separate module
            excludedPackages = [ "./lambda" ];

            ldflags = [
              "-s"
              "-w"
              "-X main.version=${version}"
              "-X main.commit=${src.rev or "unknown"}"
              "-X main.date=1970-01-01T00:00:00Z"
            ];

            meta = with pkgs.lib; {
              description = "Command-line tool for monitoring website uptime and performance";
              homepage = "https://github.com/Owloops/updo";
              license = licenses.mit;
              maintainers = [ ];
              mainProgram = "updo";
            };
          };

          updo = self.packages.${system}.default;
        }
      );

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/updo";
        };

        updo = self.apps.${system}.default;
      });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gopls
              gotools
              go-tools
            ];

            shellHook = ''
              echo "Updo development environment"
              echo "Go version: $(go version)"
            '';
          };
        }
      );
    };
}
