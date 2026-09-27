context("Create secrets", function () {
  beforeEach(function() {
    cy.visit("/");
  });

  it("Should not show noscript content", function() {
    cy.contains("Cryptopgrahy is performed in the browser").not();
  });

  it("Never serializes the secret into a native form submission", function() {
    cy.get("#secret").should("not.have.attr", "name");
  });

  it("Does not leak the secret into the URL when the form is submitted natively", function() {
    cy.get("#secret").type("hunter2");
    // form.submit() bypasses the onsubmit handler, just like a submit before
    // keedrop.js has loaded
    cy.get("#store-form").then(function($form) {
      $form[0].submit();
    });
    // A native GET submit always navigates to a URL with a "?", wait for it
    cy.location("href").should("include", "?").and("not.contain", "hunter2");
  });

  describe("API Server errors", function() {
    beforeEach(function() {
      cy.intercept("/api/secret", { forceNetworkError: true });
    });

    it("API Server not responsive", function() {
      // does not work cy.focused().should("have.id", "secret")
      cy.get("#secret").type("Test").should("have.value", "Test");
      cy.contains("Encrypt").click();
      cy.contains("Could not connect");
    });
  });

  describe("Secret size limit", function() {
    it("Rejects a secret over 46 KB without contacting the server", function() {
      cy.intercept("/api/secret", { mnemo: "deadbead" }).as("postSecret");
      cy.get("#secret").invoke("val", "x".repeat(46 * 1024 + 1));
      cy.contains("Encrypt").click();
      cy.contains("The secret is too large");
      cy.get("@postSecret.all").should("have.length", 0);
    });

    it("Shows the size error when the server answers 413", function() {
      cy.intercept("/api/secret", { statusCode: 413 });
      cy.get("#secret").type("Test");
      cy.contains("Encrypt").click();
      cy.contains("The secret is too large");
      cy.get("#secret").should("be.visible");
    });
  });

  describe("Encrypt", function() {
    beforeEach(function() {
      cy.intercept("/api/secret", { mnemo: "deadbead" }).as("postSecret");
    });

    it("Sends the secret to the API on the same origin as the page", function() {
      cy.get("#secret").type("Test");
      cy.contains("Encrypt").click();
      cy.wait("@postSecret").then(function(interception) {
        cy.location("origin").should("eq", new URL(interception.request.url).origin);
      });
    });

    it("API server generates a secret", function() {
      cy.get("#secret").type("Test").should("have.value", "Test");
      cy.contains("Encrypt").click();
      cy.contains("Send this link");
    });

    it("Copy text and encrypt another should reset copy button text", function() {
      // The Clipboard API rejects unless the window is focused, which headless
      // CI runs do not guarantee, so make copying deterministic
      cy.window().then(win => {
        if (win.navigator.clipboard) {
          cy.stub(win.navigator.clipboard, "writeText").resolves();
        }
      });
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
      cy.get("#secret").type("Test").should("have.value", "Test");
      cy.contains("Encrypt").click();
      cy.wait("@postSecret");
      // Only click once the link is rendered and the result row has finished
      // revealing, otherwise the click can race the result being shown
      cy.get("#result-box").should("not.have.value", "");
      cy.get("#copy").should("be.visible").click();
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
