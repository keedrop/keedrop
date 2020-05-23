context("Create secrets", function () {
  beforeEach(function() {
    cy.visit("/");
  });

  it("Should not show noscript content", function() {
    cy.contains("The secret is encrypted locally").not();
  });

  describe("API Server errors", function() {
    before(function() {
      cy.server({ force404: true });
      cy.route({
        method: "POST",
        url: "/api/secret",
        response: { mnemo: "deadbeadd00de"}
      });
    });

    it("API Server not responsive", function() {
      // does not work cy.focused().should("have.id", "secret")
      cy.get("#secret").type("Test").should("have.value", "Test");
      cy.contains("Encrypt").click();
      cy.contains("Could not connect");
    });
  });

  describe("Encrypt", function() {
    before(function() {
      cy.server({ force404: true });
      cy.route({
        method: "POST",
        url: "/api/secret",
        response: { mnemo: "deadbeadd00de"}
      });
    });

    it("API server generates a secret", function() {
      cy.get("#secret").type("Test").should("have.value", "Test");
      cy.contains("Encrypt").click();
      cy.contains("Send this link");
    });

    it("should submit the form on press of ENTER", function() {
      cy.get("#secret").type("Test{enter}");
      cy.contains("Send this link");
    });
  });
});
