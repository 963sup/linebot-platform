// accountLinkFromConnection maps the existing technical persistence DTO into
// the domain model without leaking OAuth credentials or Firestore metadata.
// The legacy wire/storage representation stays unchanged during migration.
func accountLinkFromConnection(conn outbound.OAuthConnection) (aggregates.AccountLink, error) {
	link, err := factories.NewVerifiedLink(conn.OwnerID, conn.GoogleSubject, conn.Scopes)
	if err != nil {
		return aggregates.AccountLink{}, err
	}
	if !conn.Active {
		if err := link.Disconnect(); err != nil {
			return aggregates.AccountLink{}, err
		}
	}
	return link, nil
}
