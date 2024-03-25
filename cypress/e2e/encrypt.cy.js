context("Create secrets", function () {
  beforeEach(function() {
    cy.visit("/");
  });

  it("Should not show noscript content", function() {
    cy.contains("Cryptopgrahy is performed in the browser").not();
  });

  describe("API Server errors", function() {
    before(function() {
      cy.intercept({ force404: true });

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
      cy.intercept("https://keedrop.com/api/secret",  { mnemo: "deadbead" }).as("postSecret");
    });

    it("API server generates a secret", function() {
      cy.get("#secret").type("Test").should("have.value", "Test");
      cy.contains("Encrypt").click();
      cy.contains("Send this link");
    });

    it.only("Copy text and encrypt another should reset copy button text", function() {
      cy.get("#secret").type("Test").should("have.value", "Test");
      cy.contains("Encrypt").click();
      cy.wait("@postSecret");
      // Monkeypatch execCommand("copy") since cypress can't send native events
      // and copy can be only executed when triggered by native event
      cy.document().then( doc => {
        const old = doc.execCommand;
        doc.execCommand = (commandId, showUI, value) => {
          if (commandId === "copy") {
            return true;
          } else {
            return old(commandId, showUI, value);
          }
        };
      });
      cy.get("#copy").click();
      cy.get("#copy").should("contain", "Copied!");

      cy.get("#secret").type("2");
      cy.contains("Encrypt").click();
      cy.wait("@postSecret");
      cy.get("#copy").should("not.contain", "Copied!");
    });

    it("should submit the form on press of ENTER", function() {
      cy.get("#secret").type("Test{enter}");
      cy.contains("Send this link");
    });
  });
});
