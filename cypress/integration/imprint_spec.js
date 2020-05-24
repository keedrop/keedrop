context("Imprint page", function () {
  describe("common infos on all imprint pages", function() {
    [
      "/de/imprint",
      "/imprint"
    ].forEach(url => {
      it("contains the GPG key on " + url, function() {
        cy.visit(url);
        cy.contains("GPG Key").should("exist");
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
      cy.contains("Contact in compliance with german law:").should("exist");
    });
  });

});
