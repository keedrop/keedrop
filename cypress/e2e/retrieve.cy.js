describe("Imprint page", function () {
  it("detect tampered or invalid link", function() {
    cy.visit("/r/#deadbeadd00de");
    cy.get("#decrypt").click();
    cy.contains("It looks like someone tampered with your link.");
  });
});

describe("Retrieve secrets", function () {
  it("Fetches the secret from the API on the same origin as the page", function() {
    cy.intercept("GET", "/api/secret/*", { statusCode: 404, body: {} }).as("getSecret");
    // A syntactically valid link: mnemo and a 32 byte key
    cy.visit("/r/#deadbead_" + "A".repeat(43) + "=");
    cy.get("#decrypt").click();
    cy.wait("@getSecret").then(function(interception) {
      cy.location("origin").then(function(origin) {
        expect(interception.request.url).to.eq(origin + "/api/secret/deadbead");
      });
    });
  });
});
