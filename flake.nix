{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";
  };

  outputs = { self, nixpkgs }:
    with import nixpkgs { system = "x86_64-linux"; };
    let pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in {
      devShell.x86_64-linux = pkgs.mkShell {
        buildInputs = [ pkgs.go_1_19 pkgs.gnumake pkgs.gopls pkgs.gotools pkgs.efm-langserver ];
      };
      formatter.x86_64-linux = pkgs.nixpkgs-fmt;
    };
}
