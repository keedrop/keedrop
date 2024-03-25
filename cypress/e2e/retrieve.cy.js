describe("Imprint page", function () {
  it("detect tampered or invalid link", function() {
    cy.visit("/r/#deadbeadd00de");
    cy.get("#decrypt").click();
    cy.contains("It looks like someone tampered with your link.");
  });
});
