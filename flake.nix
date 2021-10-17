{
  description = "Encode in a character set of your choice";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachDefaultSystem (system: let
    pkgs = nixpkgs.legacyPackages.${system};
  in rec {
    packages.aces = pkgs.buildGoModule {
      name = "aces";
      src = ./.;
      vendorSha256 = null;
      meta = with pkgs.lib; {
        description = "Encode in a character set of your choice";
        homepage = "https://github.com/quackduck/aces";
        license = licenses.mit;
        platforms = platforms.linux ++ platforms.darwin;
      };
    };
    defaultPackage = packages.aces;
  });
}
