context("Imprint page", function () {
  describe("common infos on all imprint pages", function() {
    [
      "/de/imprint",
      "/imprint"
    ].forEach(url => {
      it("contains the GPG key on " + url, function() {
        cy.visit(url);
        cy.contains("E5FF731E").should("exist");
      });
    });
  });

  describe("region specific informations", function() {
    it("should have required infos on German page", function() {
      cy.visit("/de/imprint/");
      cy.contains("Anbieter im Sinne des TDG/MDStV:").should("exist");
    });

    it("should have translated infos on non German page", function() {
      cy.visit("/imprint/");
      cy.contains("Contact according with German law:").should("exist");
    });
  });

});
