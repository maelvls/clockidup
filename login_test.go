package main

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/maelvls/clockidup/clockify"
	"github.com/maelvls/clockidup/mocks"
	"github.com/stretchr/testify/assert"
)

func Test_checkToken(t *testing.T) {
	tests := []struct {
		name       string
		givenToken string
		givenMock  func(*mocks.MockclockifyClientMockRecorder)
		want       bool
		wantErr    error
	}{
		{
			givenToken: "valid-token",
			givenMock: func(mock *mocks.MockclockifyClientMockRecorder) {
				mock.Workspaces().Return(nil, nil)
			},
			want: true,
		},
		{
			givenToken: "invalid-token",
			givenMock: func(mock *mocks.MockclockifyClientMockRecorder) {
				mock.Workspaces().Return(nil, clockify.ErrClockify{
					Message: "Full authentication is required to access this resource", Code: 1000, Status: 401,
				})
			},
			want: false,
		},
		{
			givenToken: "valid-token-but-unauthorized",
			givenMock: func(mock *mocks.MockclockifyClientMockRecorder) {
				mock.Workspaces().Return(nil, clockify.ErrClockify{
					Message: "", Status: 403,
				})
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.givenToken, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cl := mocks.NewMockclockifyClient(ctrl)
			tt.givenMock(cl.EXPECT())

			got, gotErr := checkToken(tt.givenToken, func(token string) clockifyClient {
				assert.Equal(t, tt.givenToken, token)
				return cl
			})
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantErr, gotErr)
		})
	}
}
