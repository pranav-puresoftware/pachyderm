package server

import (
	"github.com/pachyderm/pachyderm/src/client"
	"github.com/pachyderm/pachyderm/src/client/auth"
	"github.com/pachyderm/pachyderm/src/client/pfs"
	"github.com/pachyderm/pachyderm/src/client/pkg/errors"
	"github.com/pachyderm/pachyderm/src/client/pkg/grpcutil"
	"github.com/pachyderm/pachyderm/src/server/pkg/serviceenv"
	txnenv "github.com/pachyderm/pachyderm/src/server/pkg/transactionenv"
	"golang.org/x/net/context"
)

var _ APIServer = &valAPIServer{}

type valAPIServer struct {
	APIServer
	env *serviceenv.ServiceEnv
}

func newValidated(inner APIServer, env *serviceenv.ServiceEnv) *valAPIServer {
	return &valAPIServer{
		APIServer: inner,
		env:       env,
	}
}

// DeleteRepoInTransaction is identical to DeleteRepo except that it can run
// inside an existing etcd STM transaction.  This is not an RPC.
func (a *valAPIServer) DeleteRepoInTransaction(
	txnCtx *txnenv.TransactionContext,
	request *pfs.DeleteRepoRequest,
) error {
	repo := request.Repo
	// Check if the caller is authorized to delete this repo
	if err := a.checkIsAuthorizedInTransaction(txnCtx, repo, auth.Scope_OWNER); err != nil {
		return err
	}
	return a.APIServer.DeleteRepoInTransaction(txnCtx, request)
}

// DeleteCommitInTransaction is identical to DeleteCommit except that it can run
// inside an existing etcd STM transaction.  This is not an RPC.
func (a *valAPIServer) DeleteCommitInTransaction(
	txnCtx *txnenv.TransactionContext,
	request *pfs.DeleteCommitRequest,
) error {
	userCommit := request.Commit
	// Validate arguments
	if userCommit == nil {
		return errors.New("commit cannot be nil")
	}
	if userCommit.Repo == nil {
		return errors.New("commit repo cannot be nil")
	}
	if err := a.checkIsAuthorizedInTransaction(txnCtx, userCommit.Repo, auth.Scope_WRITER); err != nil {
		return err
	}
	return a.APIServer.DeleteCommitInTransaction(txnCtx, request)
}

func (a *valAPIServer) getAuth(ctx context.Context) client.AuthAPIClient {
	return a.env.GetPachClient(ctx)
}

func (a *valAPIServer) checkIsAuthorized(ctx context.Context, r *pfs.Repo, s auth.Scope) error {
	client := a.getAuth(ctx)
	me, err := client.WhoAmI(ctx, &auth.WhoAmIRequest{})
	if auth.IsErrNotActivated(err) {
		return nil
	}
	req := &auth.AuthorizeRequest{Repo: r.Name, Scope: s}
	resp, err := client.Authorize(ctx, req)
	if err != nil {
		return errors.Wrapf(grpcutil.ScrubGRPC(err), "error during authorization check for operation on \"%s\"", r.Name)
	}
	if !resp.Authorized {
		return &auth.ErrNotAuthorized{Subject: me.Username, Repo: r.Name, Required: s}
	}
	return nil
}

func (a *valAPIServer) checkIsAuthorizedInTransaction(txnCtx *txnenv.TransactionContext, r *pfs.Repo, s auth.Scope) error {
	me, err := txnCtx.Client.WhoAmI(txnCtx.ClientContext, &auth.WhoAmIRequest{})
	if auth.IsErrNotActivated(err) {
		return nil
	}

	req := &auth.AuthorizeRequest{Repo: r.Name, Scope: s}
	resp, err := txnCtx.Auth().AuthorizeInTransaction(txnCtx, req)
	if err != nil {
		return errors.Wrapf(grpcutil.ScrubGRPC(err), "error during authorization check for operation on \"%s\"", r.Name)
	}
	if !resp.Authorized {
		return &auth.ErrNotAuthorized{Subject: me.Username, Repo: r.Name, Required: s}
	}
	return nil
}
