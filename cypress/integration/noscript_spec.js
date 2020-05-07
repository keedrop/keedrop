describe("noscript", function () {
  it("should show normally when script is enabled", function() {
    debugger
    cy.visit("/")
    cy.contains("The password is encrypted locally and for that you need to enable JavaScript.").should("not.visible")
    cy.contains("Please enable JavaScript").should("not.visible")
    cy.get("#encrypt").should("visible").should("be.enabled")
  })

  it("should show noscript banner when script is disabled", function() {
      debugger
      cy.visit("/", { script: false })
      cy.contains("The password is encrypted locally and for that you need to enable JavaScript.").should("exist")
      cy.get('button')
        .first()
        .contains("Please enable JavaScript")
        .should('be.disabled')
      cy.get("#encrypt").should("not.visible")
  })
});
