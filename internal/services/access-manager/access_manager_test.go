/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package accessmanager

import (
	"context"
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hpe/access-manager/internal/services/common"
	"github.com/hpe/access-manager/internal/services/metadata"
	"github.com/hpe/access-manager/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"regexp"
	"testing"
)

var (
	appMetrics = metrics.NewMetrics()
	started    = false
)

func initialize() {
	if !started {
		started = true
		go appMetrics.StartMetricsServer("11822")
	}
}

func checkBadAce(ms metadata.MetaStore) {
	tx, err := ms.GetTree(context.Background())
	if err != nil {
		panic(err)
	}
	k := 0
	err = tx.Walk(metadata.TOP_DOWN, func(tz *metadata.MetaTree) error {
		for _, ann := range tz.Meta {
			switch ann.Tag {
			case "ace":
				k++
				_, err := ann.AsACE()
				if err != nil {
					return err
				}
			case "applied-role":
				_, err := ann.AsAppliedRole()
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	fmt.Printf("checkBadAce: %d\n", k)
	if err != nil {
		panic(err)
	}
}

func Test_DeletePrincipal(t *testing.T) {
	initialize()
	testStore, err := metadata.OpenTestStore("new_sample")
	checkBadAce(testStore)
	assert.NoError(t, err)
	meta := NewPermissionLogic(testStore)
	pm := NewAccessManager(meta, appMetrics)

	response, err := pm.DeleteObject(context.Background(), &DeleteObjectRequest{
		Path:      "am://user/hpe/bu1/tom",
		Recursive: false,
		CallerId:  "am://user/hpe/bu1/bob",
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(1), response.Error.Error)
	assert.Contains(t, response.Error.Message, "not found")
	checkBadAce(testStore)

	details, _, err := pm.meta.GetDetails(ctx, "am://user/hpe/bu1/bob", false, "am://user/hpe/bu1/alice")
	assert.NoError(t, err)
	assert.NotNil(t, details)
	checkBadAce(testStore)

	response, err = pm.DeleteObject(context.Background(), &DeleteObjectRequest{
		Path:      "am://user/hpe/bu1/bob",
		Recursive: false,
		CallerId:  "am://user/hpe/bu1/alice",
	})
	assert.NoError(t, err)
	assert.Equal(t, int32(0), response.Error.Error)
	checkBadAce(testStore)

	exists, err := pm.meta.Exists(ctx, "am://user/hpe/bu1/bob", "am://user/hpe/bu1/alice")
	assert.NoError(t, err)
	assert.False(t, exists)
}

//	func TestPermissionManager_DeletePrincipal(t *testing.T) {
//		testCases := []struct {
//			name             string
//			deleteRequest    *DeletePrincipalRequest
//			expectedError    bool
//			expectedErrMsg   string
//			expectedResponse *DeletePrincipalResponse
//		}{
//			{
//				name: "Should Successfully delete the principal",
//				deleteRequest: &DeletePrincipalRequest{
//					Path:     "am://user/hpe/bu1/bob",
//					CallerId: "am://user/hpe/bu1/alice",
//				},
//				expectedError:  false,
//				expectedErrMsg: "",
//				expectedResponse: &DeletePrincipalResponse{
//					Error: &Status{
//						Error: 0,
//					},
//				},
//			},
//			{
//				name: "Should return error with proper error code if the DeletePrincipal principal fails",
//				deleteRequest: &DeletePrincipalRequest{
//					Path:     "am://user/hpe/bu1/tom",
//					CallerId: "am://user/hpe/bu1/alice",
//				},
//				expectedError:  true,
//				expectedErrMsg: "object doesn't exist: am://user/hpe/bu1/tom",
//				expectedResponse: &DeletePrincipalResponse{
//					Error: &Status{
//						Error:   1,
//						Message: "object doesn't exist: am://user/hpe/bu1/tom",
//					},
//				},
//			},
//			{
//				name: "Should return error if insufficient privileges",
//				deleteRequest: &DeletePrincipalRequest{
//					Path:     "am://user/hpe/bu1/alice",
//					CallerId: "am://user/hpe/bu1/bob",
//				},
//				expectedError:  true,
//				expectedErrMsg: "not allowed to administer am://user/hpe/bu1/alice",
//				expectedResponse: &DeletePrincipalResponse{
//					Error: &Status{
//						Error:   1,
//						Message: "not allowed to administer am://user/hpe/bu1/alice",
//					},
//				},
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				pc := store.SampleMetaDataOrBust(true)
//				meta := PermissionLogicManager{
//					ms: pc,
//				}
//				pm := NewAccessManager(&meta, appMetrics)
//
//				path := tc.deleteRequest.Path
//				_, before := sibs(t, pm, tc.deleteRequest.CallerId, path)
//
//				// Call the CreatePrincipal method
//				response, err := pm.DeletePrincipal(context.Background(), tc.deleteRequest)
//				assert.Nil(t, err)
//				_, siblings := sibs(t, pm, tc.deleteRequest.CallerId, path)
//				// Check if there's an error if expectedError is true
//				if tc.expectedError {
//					assert.True(t, response.Error.Error > int32(0))
//					assert.True(t, !slices.Contains(before, path) || slices.Contains(siblings, path))
//					assert.Equal(t, tc.expectedErrMsg, response.Error.Message, "Error message mismatch")
//				} else {
//					assert.False(t, slices.Contains(siblings, path))
//				}
//			})
//		}
//	}
//
//	func TestPermissionManager_GetRoles(t *testing.T) {
//		testCases := []struct {
//			name             string
//			getRolesRequest  *GetRolesRequest
//			expectedError    bool
//			expectedErrMsg   string
//			expectedResponse *GetRolesResponse
//		}{
//			{
//				name: "Should Successfully return the roles",
//				getRolesRequest: &GetRolesRequest{
//					Path:     "am://user/hpe/bu1/alice",
//					CallerId: "am://user/hpe/bu1/alice",
//				},
//				expectedError:  false,
//				expectedErrMsg: "principal am://user/hpe/bu1/tom does not exist",
//				expectedResponse: &GetRolesResponse{
//					Roles: []*AppliedRole{
//						{
//							Key:     "am://user/hpe/bu1/alice",
//							Version: 3,
//							Role:    "am://role/hpe/bu1/bu1-admin",
//						},
//					},
//					Error: &Status{
//						Error: 0,
//					},
//				},
//			},
//			{
//				name: "Should return proper error code if get roles returns error",
//				getRolesRequest: &GetRolesRequest{
//					Path:     "am://user/hpe/bu1/invisible-man",
//					CallerId: "am://user/hpe/bu1/bob",
//				},
//				expectedError:  false,
//				expectedErrMsg: "principal am://user/hpe/bu1/invisible-man not found",
//				expectedResponse: &GetRolesResponse{
//					Error: &Status{
//						Error:   1,
//						Message: "principal am://user/hpe/bu1/invisible-man not found",
//					},
//				},
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				pc := store.SampleMetaDataOrBust(true)
//				meta := PermissionLogicManager{
//					ms: pc,
//				}
//				pm := NewAccessManager(&meta, appMetrics)
//
//				// Call the CreatePrincipal method
//				response, err := pm.GetRoles(context.Background(), tc.getRolesRequest)
//				assert.Nil(t, err)
//
//				// Check if there's an error if expectedError is true
//				if tc.expectedError {
//					assert.Equal(t, tc.expectedErrMsg, response.Error.Message, "Error message mismatch")
//				}
//				assert.Equal(t, common.CleanJson(tc.expectedResponse), common.CleanJson(response))
//			})
//		}
//	}
//
// // func TestPermissionManager_UpdateRoles(t *testing.T) {
// //	testCases := []struct {
// //		name               string
// //		updateRolesRequest *ApplyRolesRequest
// //		expectedError      bool
// //		expectedErrMsg     string
// //		updateMocks        func(mps *mock_store.MockPermissionStore)
// //		expectedResponse   *ApplyRolesResponse
// //	}{
// //		{
// //			name: "Should Successfully update the role",
// //			updateRolesRequest: &ApplyRolesRequest{
// //				Path:     "am://user/hpe/bu1/tom",
// //				CallerID: "am://user/hpe/bu1/alice",
// //				AppliedRole: &AppliedRole{
// //					Path:     "am://user/hpe/bu1/tom",
// //					Version: 1,
// //					Role:    "",
// //				},
// //			},
// //			updateMocks: func(mps *mock_store.MockPermissionStore) {
// //				mps.EXPECT().
// //					GetPrincipal(ctx, gomock.Any()).
// //					Return(&metadata.PrincipalInfo{}, nil).AnyTimes()
// //
// //				mps.EXPECT().
// //					GetRoles(ctx, gomock.Any()).
// //					Return(&metadata.AppliedRole{
// //						AppliedRole: []string{},
// //					}).AnyTimes()
// //
// //				mps.EXPECT().
// //					GetPermissions(ctx, gomock.Any()).
// //					Return(&metadata.PermissionInfo{}).AnyTimes()
// //
// //				mps.EXPECT().
// //					PutRoles(ctx, gomock.Any()).
// //					Return(uint64(1), nil).Times(1)
// //
// //			},
// //			expectedError:  false,
// //			expectedErrMsg: "",
// //			expectedResponse: &UpdateRolesResponse{
// //				Version: 1,
// //				Error: &Status{
// //					Error: 0,
// //				},
// //			},
// //		},
// //		{
// //			name: "Should return proper error code if update roles returns error",
// //			updateRolesRequest: &UpdateRolesRequest{
// //				Path:     "am://user/hpe/bu1/tom",
// //				CallerID: "am://user/hpe/bu1/alice",
// //				Version:  1,
// //			},
// //			updateMocks: func(mps *mock_store.MockPermissionStore) {
// //				mps.EXPECT().
// //					GetPrincipal(ctx, gomock.Any()).
// //					Return(&metadata.PrincipalInfo{}, nil).AnyTimes()
// //
// //				mps.EXPECT().
// //					GetRoles(ctx, gomock.Any()).
// //					Return(&metadata.AppliedRole{
// //						Roles: []string{},
// //					}).AnyTimes()
// //
// //				mps.EXPECT().
// //					GetPermissions(ctx, gomock.Any()).
// //					Return(&metadata.PermissionInfo{}).AnyTimes()
// //
// //				mps.EXPECT().
// //					PutRoles(ctx, gomock.Any()).
// //					Return(uint64(1), errors.New("some error")).Times(1)
// //
// //			},
// //			expectedError:  false,
// //			expectedErrMsg: "",
// //			expectedResponse: &UpdateRolesResponse{
// //				Error: &Status{
// //					Error:   1,
// //					Message: "some error",
// //				},
// //			},
// //		},
// //	}
// //
// //	for _, tc := range testCases {
// //		t.Run(tc.name, func(t *testing.T) {
// //			ctrl := gomock.NewController(t)
// //			defer ctrl.Finish()
// //
// //			mockPC := mock_store.NewMockPermissionStore(ctrl)
// //
// //			// Create a ConcreteAccessManager with a mock PermissionLogicManager
// //			mockMeta := PermissionLogicManager{
// //				ms: mockPC,
// //			}
// //			tc.updateMocks(mockPC)
// //			pm := NewAccessManager(&mockMeta, appMetrics)
// //
// //			// Call the CreatePrincipal method
// //			response, err := pm.UpdateRoles(context.Background(), tc.updateRolesRequest)
// //
// //			// Check if there's an error if expectedError is true
// //			if tc.expectedError {
// //				assert.NotNil(t, err, "Expected an error")
// //				assert.Equal(t, tc.expectedErrMsg, err.Error(), "Error message mismatch")
// //			}
// //			assert.Equal(t, tc.expectedResponse, response)
// //		})
// //	}
// //}
//
//	func TestPermissionManager_GetChildren(t *testing.T) {
//		//goland:noinspection GoDeprecation
//		testCases := []struct {
//			name               string
//			getChildrenRequest *GetChildrenRequest
//			expectedError      bool
//			expectedErrMsg     string
//			expectedResponse   *GetChildrenResponse
//		}{
//			{
//				name: "Should Successfully return the children",
//				getChildrenRequest: &GetChildrenRequest{
//					Path:     "am://user/hpe/bu1",
//					CallerId: "am://user/hpe/bu1/alice",
//				},
//				expectedError:  false,
//				expectedErrMsg: "",
//				expectedResponse: &GetChildrenResponse{
//					Children: []string{
//						"am://user/hpe/bu1/alice", "am://user/hpe/bu1/bob",
//						"am://user/hpe/bu1/invisible-man", "am://user/hpe/bu1/x"},
//					ChildrenDetails: []*NodeDetails{
//						{
//							Path: "am://user/hpe/bu1/alice",
//							Roles: []*AppliedRole{
//								{
//									Key:     "am://user/hpe/bu1/alice",
//									Version: 3,
//									Role:    "am://role/hpe/bu1/bu1-admin",
//								},
//							},
//							InheritedRoles: []*AppliedRole{
//								{
//									Key:     "am://user/hpe",
//									Version: 3,
//									Role:    "am://role/hpe/hpe-user",
//								},
//								{
//									Key:     "am://user/hpe/bu1",
//									Version: 3,
//									Role:    "am://role/hpe/bu1/bu1-user",
//								},
//							},
//							Aces: []*ACE{},
//							InheritedAces: []*ACE{
//								{
//									Op:      Operation_VIEW,
//									Unique:  79234067,
//									Local:   false,
//									Version: 3,
//									Expiry:  0,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/hpe-user"},
//										},
//									},
//								},
//								{
//									Op:      Operation_ADMIN,
//									Unique:  3547610,
//									Local:   false,
//									Version: 3,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//										},
//									},
//								},
//							},
//							IsDirectory: false,
//						},
//						{
//							Path:  "am://user/hpe/bu1/bob",
//							Roles: []*AppliedRole{},
//							InheritedRoles: []*AppliedRole{
//								{
//									Key:     "am://user/hpe",
//									Version: 3,
//									Role:    "am://role/hpe/hpe-user",
//								},
//								{
//									Key:     "am://user/hpe/bu1",
//									Version: 3,
//									Role:    "am://role/hpe/bu1/bu1-user",
//								},
//							},
//							Aces: []*ACE{},
//							InheritedAces: []*ACE{
//								{
//									Op:      Operation_VIEW,
//									Unique:  79234067,
//									Local:   false,
//									Version: 3,
//									Expiry:  0,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/hpe-user"},
//										},
//									},
//								},
//								{
//									Op:      Operation_ADMIN,
//									Unique:  3547610,
//									Local:   false,
//									Version: 3,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//										},
//									},
//								},
//							},
//							IsDirectory: false,
//						},
//						{
//							Path: "am://user/hpe/bu1/invisible-man",
//							Roles: []*AppliedRole{
//								{
//									Key:     "am://user/hpe/bu1/invisible-man",
//									Role:    "am://role/hpe/bu1/bu1-admin",
//									Version: 3,
//								},
//							},
//							InheritedRoles: []*AppliedRole{
//								{
//									Key:     "am://user/hpe",
//									Version: 3,
//									Role:    "am://role/hpe/hpe-user",
//								},
//								{
//									Key:     "am://user/hpe/bu1",
//									Version: 3,
//									Role:    "am://role/hpe/bu1/bu1-user",
//								},
//							},
//							Aces: []*ACE{
//								{
//									Op:      Operation_VIEW,
//									Unique:  8453087,
//									Version: 3,
//									Local:   false,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//										},
//									},
//								},
//							},
//							InheritedAces: []*ACE{
//								{
//									Op:      Operation_VIEW,
//									Unique:  79234067,
//									Local:   false,
//									Version: 3,
//									Expiry:  0,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/hpe-user"},
//										},
//									},
//								},
//								{
//									Op:      Operation_ADMIN,
//									Unique:  3547610,
//									Version: 3,
//									Local:   false,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//										},
//									},
//								},
//							},
//							IsDirectory: false,
//						},
//						{
//							Path:  "am://user/hpe/bu1/x",
//							Roles: []*AppliedRole{},
//							InheritedRoles: []*AppliedRole{
//								{
//									Key:     "am://user/hpe",
//									Version: 3,
//									Role:    "am://role/hpe/hpe-user",
//								},
//								{
//									Key:     "am://user/hpe/bu1",
//									Version: 3,
//									Role:    "am://role/hpe/bu1/bu1-user",
//								},
//							},
//							Aces: []*ACE{},
//							InheritedAces: []*ACE{
//								{
//									Op:      Operation_VIEW,
//									Unique:  79234067,
//									Local:   false,
//									Version: 3,
//									Expiry:  0,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/hpe-user"},
//										},
//									},
//								},
//								{
//									Op:      Operation_ADMIN,
//									Unique:  3547610,
//									Local:   false,
//									Version: 3,
//									Permissions: []*ACL{
//										{
//											Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//										},
//									},
//								},
//							},
//							IsDirectory: true,
//						},
//					},
//					Error: &Status{
//						Error: 0,
//					},
//				},
//			},
//			{
//				name: "Should return proper error code if update roles returns error",
//				getChildrenRequest: &GetChildrenRequest{
//					Path:     "c1",
//					CallerId: "am://user/hpe/bu1/alice",
//				},
//				expectedError:  false,
//				expectedErrMsg: "",
//				expectedResponse: &GetChildrenResponse{
//					Error: &Status{
//						Error:   1,
//						Message: "invalid path c1, must start with 'am://'",
//					},
//				},
//			},
//		}
//
//		// Create a ConcreteAccessManager with sample data
//		meta := PermissionLogicManager{
//			ms: store.SampleMetaDataOrBust(true),
//		}
//		pm := NewAccessManager(&meta, appMetrics)
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//
//				response, err := pm.GetChildren(context.Background(), tc.getChildrenRequest)
//
//				// Check if there's an error if expectedError is true
//				if tc.expectedError {
//					assert.NotNil(t, err, "Expected an error")
//					assert.Equal(t, tc.expectedErrMsg, err.Error(), "Error message mismatch")
//				}
//				assert.Equal(t, common.CleanJson(tc.expectedResponse), common.CleanJson(response))
//			})
//		}
//	}
//
// // The CreateRole function is not completely implemented, hence writing only success tests
//
//	func TestPermissionManager_CreateRoles(t *testing.T) {
//		testCases := []struct {
//			name               string
//			createRolesRequest *CreateRoleRequest
//			expectedError      bool
//			expectedErrMsg     string
//			updateMocks        func(mps *mock_store.MockPermissionStore)
//			expectedResponse   *CreateRoleResponse
//		}{
//			{
//				name: "Should create role and respond back",
//				createRolesRequest: &CreateRoleRequest{
//					Path:     "am://role/hpe/bu1/bu-specific-role",
//					CallerId: "am://user/hpe/bu1/alice",
//				},
//				updateMocks:    func(mps *mock_store.MockPermissionStore) {},
//				expectedError:  false,
//				expectedErrMsg: "",
//				expectedResponse: &CreateRoleResponse{
//					Version: 2,
//					Error: &Status{
//						Error: 0,
//					},
//				},
//			},
//			{
//				name: "Should fail to create duplicate role",
//				createRolesRequest: &CreateRoleRequest{
//					Path:     "am://role/hpe/bu1/bu1-admin",
//					CallerId: "am://user/hpe/bu1/alice",
//				},
//				updateMocks:    func(mps *mock_store.MockPermissionStore) {},
//				expectedError:  false,
//				expectedErrMsg: "",
//				expectedResponse: &CreateRoleResponse{
//					Version: 0,
//					Error: &Status{
//						Error:   1,
//						Message: "version mismatch: object am://role/hpe/bu1/bu1-admin already exists, cannot create",
//					},
//				},
//			},
//			{
//				name: "Should fail to create role without sufficient permission",
//				createRolesRequest: &CreateRoleRequest{
//					Path:     "am://role/hpe/bu1/bu1-special",
//					CallerId: "am://user/hpe/bu1/bob",
//				},
//				updateMocks:    func(mps *mock_store.MockPermissionStore) {},
//				expectedError:  false,
//				expectedErrMsg: "",
//				expectedResponse: &CreateRoleResponse{
//					Version: 0,
//					Error: &Status{
//						Error:   1,
//						Message: "user not permitted to create dataset",
//					},
//				},
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				// Create a ConcreteAccessManager
//				meta := PermissionLogicManager{
//					ms: store.SampleMetaDataOrBust(true),
//				}
//				pm := NewAccessManager(&meta, appMetrics)
//
//				// Call the CreateRole method
//				response, err := pm.CreateRole(context.Background(), tc.createRolesRequest)
//
//				// Check if there's an error if expectedError is true
//				if tc.expectedError {
//					assert.NotNil(t, err, "Expected an error")
//					assert.Equal(t, tc.expectedErrMsg, err.Error(), "Error message mismatch")
//				}
//				assert.Equal(t, tc.expectedResponse, response)
//			})
//		}
//	}
//
//	func TestPermissionManager_CreateDataset(t *testing.T) {
//		testCases := []struct {
//			name                 string
//			createDatasetRequest *CreateDatasetRequest
//			expectedError        bool
//			expectedErrMsg       string
//		}{
//			{
//				name: "Should Successfully create dataset",
//				createDatasetRequest: &CreateDatasetRequest{
//					Path:       "am://data/hpe/bu1/tom",
//					CallerId:   "am://user/hpe/bu1/alice",
//					Url:        "s3://bucket1/home",
//					UrlPattern: "s3://bucket2/home/*",
//				},
//				expectedError:  false,
//				expectedErrMsg: "",
//			},
//			{
//				name: "Should return proper error code if put dataset returns error",
//				createDatasetRequest: &CreateDatasetRequest{
//					Path:       "am://data/hpe/bu1/tom",
//					CallerId:   "am://user/hpe/bu1/bob",
//					Url:        "s3://bucket1/home",
//					UrlPattern: "s3://bucket2/home/*",
//				},
//				expectedError:  true,
//				expectedErrMsg: "user not permitted to create dataset",
//			},
//		}
//
//		for _, tc := range testCases {
//			t.Run(tc.name, func(t *testing.T) {
//				meta := PermissionLogicManager{
//					ms: store.SampleMetaDataOrBust(true),
//				}
//				pm := NewAccessManager(&meta, appMetrics)
//
//				// Call the CreatePrincipal method
//				ctx := context.Background()
//				response, err := pm.CreateDataset(ctx, tc.createDatasetRequest)
//				assert.Nil(t, err)
//
//				// Check if there's an error if expectedError is true
//				if tc.expectedError {
//					assert.True(t, response.Error.Error > 0)
//					assert.Equal(t, tc.expectedErrMsg, response.Error.Message, "Error message mismatch")
//				}
//
//				_, children := sibs(t, pm, tc.createDatasetRequest.CallerId, tc.createDatasetRequest.Path)
//				if tc.expectedError {
//					assert.False(t, slices.Contains(children, tc.createDatasetRequest.Path))
//				} else {
//					assert.True(t, slices.Contains(children, tc.createDatasetRequest.Path))
//				}
//			})
//		}
//	}
func TestPermissionManager_GetDetails(t *testing.T) {
	initialize()
	tests := []struct {
		name           string
		request        *GetDetailsRequest
		wantRoles      []string
		wantAces       []string
		inheritedRoles []string
		inheritedAces  []string
		statusError    string
		wantErr        error
	}{
		{
			name: "empty url",
			request: &GetDetailsRequest{
				Path:            "am://",
				CallerId:        "am://user/hpe/bu1/alice",
				IncludeChildren: false,
			},
			wantRoles: nil,
			wantAces: []string{
				`{"op": "ADMIN","local": true,"acls": [{"roles": ["## Redacted role ##"]}],"tag": "ace"}`,
				`{"op": "WRITE","local": true,"acls": [{"roles": ["## Redacted role ##"]}],"tag": "ace"}`,
			},
			inheritedRoles: nil,
			inheritedAces:  nil,
			wantErr:        nil,
			statusError:    "",
		},
		{
			name: "base case",
			request: &GetDetailsRequest{
				Path:     "am://user/hpe/bu1/alice",
				CallerId: "am://user/hpe/bu1/alice",
			},
			wantRoles:      []string{"am://role/hpe/bu1/bu1-admin"},
			wantAces:       nil,
			inheritedRoles: []string{"am://role/hpe/hpe-user", "am://role/hpe/bu1/bu1-user"},
			inheritedAces: []string{
				`{"op": "ADMIN","acls": [{"roles": ["am://role/hpe/bu1/bu1-admin"]}],"tag": "ace"}`,
				`{"op": "WRITE","acls": [{"roles": ["am://role/hpe/bu1/bu1-admin"]}],"tag": "ace"}`,
				`{"op": "VIEW","acls": [{"roles": ["am://role/hpe/hpe-user", "## Redacted role ##"]}],"tag": "ace"}`,
			},

			wantErr:     nil,
			statusError: "",
		},
		{
			name: "redacted base case",
			request: &GetDetailsRequest{
				Path:     "am://user/hpe/bu1/alice",
				CallerId: "am://user/hpe/bu1/bob",
			},
			wantRoles:      []string{"## Redacted role ##"},
			wantAces:       nil,
			inheritedRoles: []string{"am://role/hpe/hpe-user", "am://role/hpe/bu1/bu1-user"},
			inheritedAces: []string{
				`{"op": "ADMIN","acls": [{"roles": ["## Redacted role ##"]}],"tag": "ace"}`,
				`{"op": "WRITE","acls": [{"roles": ["## Redacted role ##"]}],"tag": "ace"}`,
				`{"op": "VIEW","acls": [{"roles": ["am://role/hpe/hpe-user", "## Redacted role ##"]}],"tag": "ace"}`,
			},
			wantErr:     nil,
			statusError: "",
		},
		{
			name: "visible man",
			request: &GetDetailsRequest{
				Path:     "am://user/hpe/bu1/invisible-man",
				CallerId: "am://user/hpe/bu1/alice",
			},
			wantRoles: []string{"am://role/hpe/bu1/bu1-admin"},
			wantAces: []string{
				`{"op": "VIEW","acls": [{"roles": ["am://role/hpe/bu1/bu1-admin", "## Redacted role ##"]}],"tag": "ace"}`,
			},
			inheritedRoles: []string{"am://role/hpe/hpe-user", "am://role/hpe/bu1/bu1-user"},
			inheritedAces: []string{
				`{"op": "ADMIN","acls": [{"roles": ["am://role/hpe/bu1/bu1-admin"]}],"tag": "ace"}`,
				`{"op": "WRITE","acls": [{"roles": ["am://role/hpe/bu1/bu1-admin"]}],"tag": "ace"}`,
				`{"op": "VIEW","acls": [{"roles": ["am://role/hpe/hpe-user", "## Redacted role ##"]}],"tag": "ace"}`,
			},
			wantErr:     nil,
			statusError: "",
		},
		{
			name: "invisible man",
			request: &GetDetailsRequest{
				Path:     "am://user/hpe/bu1/invisible-man",
				CallerId: "am://user/hpe/bu1/bob",
			},
			wantRoles:      nil,
			wantAces:       nil,
			inheritedRoles: nil,
			inheritedAces:  nil,
			wantErr:        nil,
			statusError:    "not found",
		},
	}
	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testStore, err := metadata.OpenTestStore("new_sample")
			assert.NoError(t, err)
			meta := NewPermissionLogic(testStore)
			pm := NewAccessManager(meta, appMetrics)

			got, err := pm.GetDetails(ctx, tt.request)
			if tt.wantErr == nil && err != nil {
				t.Errorf("GetDetails(%v, %v) unexpected error %s", ctx, tt.request, err)
				return
			}
			if tt.wantErr != nil && err == nil {
				t.Errorf("GetDetails(%v, %v) didn't see expected error %s", ctx, tt.request, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			if tt.statusError != "" {
				assert.Equal(t, int32(1), got.Error.Error)
				assert.Contains(t, got.Error.Message, tt.statusError)
			} else {
				assert.NoError(t, compareRoleSet(got.Details.Roles, tt.wantRoles, "direct"))
				assert.NoError(t, compareACEs(got.Details.Aces, tt.wantAces, "direct"))
				assert.NoError(t, compareRoleSet(got.Details.InheritedRoles, tt.inheritedRoles, "inherited"))
				assert.NoError(t, compareACEs(got.Details.InheritedAces, tt.inheritedAces, "inherited"))
			}

		})
	}
}

func compareRoleSet(a []*metadata.AppliedRole, b []string, message string) error {
	ax := mapset.NewSet[string]()
	for _, x := range a {
		ax.Add(x.Role)
	}
	bx := mapset.NewSet[string]()
	for _, x := range b {
		bx.Add(x)
	}
	if !ax.Equal(bx) {
		return fmt.Errorf("%s: got extra %v, missing %v", message, ax.Difference(bx), bx.Difference(ax))
	}
	return nil
}

func compareACEs(a []*metadata.ACE, b []string, message string) error {
	ax := map[string]string{}
	extraFields := regexp.MustCompile(`,\s*("unique"|"version"):\s*\d+`)
	whiteSpace := regexp.MustCompile(`\s*`)
	for _, x := range a {
		s := common.CleanJson(x)
		s = extraFields.ReplaceAllString(s, "")
		key := whiteSpace.ReplaceAllString(s, "")
		ax[key] = s
		fmt.Printf("<<%s>>\n", s)
	}
	bx := map[string]string{}
	for _, s := range b {
		s = extraFields.ReplaceAllString(s, "")
		key := whiteSpace.ReplaceAllString(s, "")
		bx[key] = s
	}
	ak := mapset.NewSetFromMapKeys(ax)
	bk := mapset.NewSetFromMapKeys(bx)
	if !ak.Equal(bk) {
		extra := values(ax, ak.Difference(bk))
		missing := values(bx, bk.Difference(ak))
		return fmt.Errorf("%s: got extra %v, missing %v", message, extra, missing)
	}
	return nil
}

func values[M interface{ ~map[K]V }, K comparable, V comparable](ax M, keys mapset.Set[K]) mapset.Set[V] {
	r := mapset.NewSet[V]()
	for x := range keys.Iter() {
		r.Add(ax[x])
	}
	return r
}

//
//func TestPermissionManager_PutDetails(t *testing.T) {
//	onlyRoles := true
//	onlyACEs := false
//	tests := []struct {
//		name          string
//		request       *PutDetailsRequest
//		outputDetails *NodeDetails
//		want          *PutDetailsResponse
//		wantErr       error
//	}{
//		{
//			name: "update only roles",
//			request: &PutDetailsRequest{
//				Path:        "am://user/hpe/bu1",
//				CallerId:    "am://user/hpe/bu1/alice",
//				UpdateRoles: &onlyRoles,
//				Details: &NodeDetails{
//					Roles: []*AppliedRole{
//						{
//							Key:     "am://user/hpe/bu1",
//							Role:    "am://role/hpe/bu1/bu1-admin",
//							Version: 0,
//						},
//					},
//					Aces: []*ACE{},
//				},
//			},
//			outputDetails: &NodeDetails{
//				Path: "am://user/hpe/bu1",
//				Roles: []*AppliedRole{
//					{
//						Key:     "am://user/hpe/bu1",
//						Role:    "am://role/hpe/bu1/bu1-user",
//						Version: 3,
//					},
//					{
//						Key:     "am://user/hpe/bu1",
//						Role:    "am://role/hpe/bu1/bu1-admin",
//						Version: 2,
//					},
//				},
//				InheritedRoles: []*AppliedRole{
//					{
//						Key:     "am://user/hpe",
//						Version: 3,
//						Role:    "am://role/hpe/hpe-user",
//					},
//				},
//				Aces: []*ACE{
//					{
//						Op:      Operation_ADMIN,
//						Unique:  3547610,
//						Version: 3,
//						Permissions: []*ACL{
//							{
//								Roles: []string{
//									"am://role/hpe/bu1/bu1-admin",
//								},
//							},
//						},
//					},
//				},
//				InheritedAces: []*ACE{
//					{
//						Op:      Operation_VIEW,
//						Unique:  79234067,
//						Version: 3,
//						Permissions: []*ACL{
//							{
//								Roles: []string{"am://role/hpe/hpe-user"},
//							},
//						},
//					},
//				},
//				IsDirectory: true,
//			},
//			want: &PutDetailsResponse{
//				Error: &Status{
//					Error:   0,
//					Message: "",
//				},
//			},
//			wantErr: nil,
//		},
//		{
//			name: "update only aces",
//			request: &PutDetailsRequest{
//				Path:        "am://user/hpe/bu1",
//				CallerId:    "am://user/hpe/bu1/alice",
//				UpdateRoles: &onlyACEs,
//				Details: &NodeDetails{
//					Roles: []*AppliedRole{
//						{
//							Key:     "am://user/hpe/bu1",
//							Role:    "am://role/hpe/bu1/bu1-admin",
//							Version: 3,
//						},
//					},
//					Aces: []*ACE{
//						{
//							Op:      Operation_VIEW,
//							Unique:  84353087,
//							Local:   false,
//							Version: 0,
//							Permissions: []*ACL{
//								{
//									Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//								},
//							},
//						},
//					},
//				},
//			},
//			outputDetails: &NodeDetails{
//				Path: "am://user/hpe/bu1",
//				Roles: []*AppliedRole{
//					{
//						Key:     "am://user/hpe/bu1",
//						Version: 3,
//						Role:    "am://role/hpe/bu1/bu1-user",
//					},
//				},
//				InheritedRoles: []*AppliedRole{
//					{
//						Key:     "am://user/hpe",
//						Version: 3,
//						Role:    "am://role/hpe/hpe-user",
//					},
//				},
//				Aces: []*ACE{
//					{
//						Op:      Operation_ADMIN,
//						Unique:  3547610,
//						Version: 3,
//						Permissions: []*ACL{
//							{
//								Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//							},
//						},
//					},
//					{
//						Op:     Operation_VIEW,
//						Unique: 1703634693574,
//						Permissions: []*ACL{
//							{
//								Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//							},
//						},
//					},
//				},
//				InheritedAces: []*ACE{
//					{
//						Op:      Operation_VIEW,
//						Unique:  79234067,
//						Version: 3,
//						Permissions: []*ACL{
//							{
//								Roles: []string{"am://role/hpe/hpe-user"},
//							},
//						},
//					},
//				},
//				IsDirectory: true,
//			},
//			want: &PutDetailsResponse{
//				Error: &Status{
//					Error:   0,
//					Message: "",
//				},
//			},
//			wantErr: nil,
//		},
//		{
//			name: "update both",
//			request: &PutDetailsRequest{
//				Path:        "am://user/hpe/bu1",
//				CallerId:    "am://user/hpe/bu1/alice",
//				UpdateRoles: nil,
//				Details: &NodeDetails{
//					Roles: []*AppliedRole{
//						{
//							Key:     "am://user/hpe/bu1",
//							Role:    "am://role/hpe/bu1/bu1-admin",
//							Version: 0,
//						},
//					},
//					Aces: []*ACE{
//						{
//							Op:      Operation_ADMIN,
//							Unique:  3547610,
//							Version: 3,
//							Local:   false,
//							Expiry:  0,
//							Permissions: []*ACL{
//								{
//									Roles: []string{"am://role/hpe/bu1/bu1-user"},
//								},
//							},
//						},
//					},
//				},
//			},
//			outputDetails: &NodeDetails{
//				Path: "am://user/hpe/bu1",
//				Roles: []*AppliedRole{
//					{
//						Key:     "am://user/hpe/bu1",
//						Version: 3,
//						Role:    "am://role/hpe/bu1/bu1-user",
//					},
//					{
//						Key:     "am://user/hpe/bu1",
//						Version: 2,
//						Role:    "am://role/hpe/bu1/bu1-admin",
//					},
//				},
//				InheritedRoles: []*AppliedRole{
//					{
//						Key:     "am://user/hpe",
//						Version: 3,
//						Role:    "am://role/hpe/hpe-user",
//					},
//				},
//				Aces: []*ACE{
//					{
//						Op:      Operation_ADMIN,
//						Version: 3,
//						Permissions: []*ACL{
//							{
//								Roles: []string{"am://role/hpe/bu1/bu1-user"},
//							},
//						},
//					},
//				},
//				InheritedAces: []*ACE{
//					{
//						Op:      Operation_VIEW,
//						Unique:  79234067,
//						Version: 3,
//						Permissions: []*ACL{
//							{
//								Roles: []string{"am://role/hpe/hpe-user"},
//							},
//						},
//					},
//				},
//				IsDirectory: true,
//			},
//			want: &PutDetailsResponse{
//				Error: &Status{
//					Error:   0,
//					Message: "",
//				},
//			},
//			wantErr: nil,
//		},
//		{
//			name: "invalid version",
//			request: &PutDetailsRequest{
//				Path:        "am://user/hpe/bu1",
//				CallerId:    "am://user/hpe/bu1/alice",
//				UpdateRoles: nil,
//				Details: &NodeDetails{
//					Roles: []*AppliedRole{
//						{
//							Key:     "am://user/hpe/bu1",
//							Role:    "am://role/hpe/bu1/bu1-admin",
//							Version: 2,
//						},
//					},
//					Aces: []*ACE{
//						{
//							Op:      Operation_VIEW,
//							Unique:  8453087,
//							Local:   false,
//							Version: 2,
//							Permissions: []*ACL{
//								{
//									Roles: []string{"am://role/hpe/bu1/bu1-admin"},
//								},
//							},
//						},
//					},
//				},
//			},
//			want: &PutDetailsResponse{
//				Error: &Status{
//					Error:   1,
//					Message: `must use version 0 or 1 to create new object "am://user/hpe/bu1"`,
//				},
//			},
//			wantErr: nil,
//		},
//	}
//	ctx := context.Background()
//	for _, tt := range tests {
//		meta := PermissionLogicManager{
//			ms: store.SampleMetaDataOrBust(true),
//		}
//		pm := NewAccessManager(&meta, appMetrics)
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := pm.PutDetails(ctx, tt.request)
//			if tt.wantErr == nil && err != nil {
//				t.Errorf("PutDetails(%v, %v) unexpected error %s", ctx, tt.request, err)
//				return
//			}
//			if tt.wantErr != nil && err == nil {
//				t.Errorf("PutDetails(%v, %v) didn't see expected error %s", ctx, tt.request, tt.wantErr)
//				return
//			}
//			if (tt.wantErr != nil) && (err != nil) && tt.wantErr.Error() != err.Error() {
//				t.Errorf("PutDetails(%v, %v) wanted %s, got %s", ctx, tt.request, tt.wantErr.Error(), err.Error())
//				return
//			}
//			bufWant, err := prototext.Marshal(tt.want)
//			if err != nil {
//				assert.Fail(t, "Couldn't convert to string")
//			}
//			bufGot, err := prototext.Marshal(got)
//			if err != nil {
//				assert.Fail(t, "Couldn't convert to string")
//			}
//			assert.Equalf(t, string(bufWant), string(bufGot), "GetDetails(%v, %v)", ctx, tt.request)
//
//			// Check if the data is updated
//			if tt.want.Error.Error == 0 {
//				details, err := pm.GetDetails(ctx, &GetDetailsRequest{
//					Path:     tt.request.Path,
//					CallerId: tt.request.CallerId,
//				})
//				if err != nil {
//					assert.Fail(t, "Couldn't do GetDetails")
//				}
//				// force the uniques to match since they are generated randomly
//				for _, ace := range details.Details.Aces {
//					ace.Unique = 0
//				}
//				for _, ace := range tt.outputDetails.Aces {
//					ace.Unique = 0
//				}
//				assert.Equal(t, common.CleanJson(tt.outputDetails), common.CleanJson(details.Details))
//			}
//		})
//	}
//}
