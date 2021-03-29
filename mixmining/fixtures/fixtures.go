// Copyright 2020 Nym Technologies SA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fixtures

import "github.com/nymtech/nym/validator/nym/models"

// MixStatusesList A list of mix statuses
func MixStatusesList() []models.PersistedMixStatus {
	booltrue := true
	m1 := models.PersistedMixStatus{
		MixStatus: models.MixStatus{
			IPVersion: "6",
			PubKey:    "pubkey1",
			Up:        &booltrue,
		},
		Timestamp: 123,
	}

	m2 := models.PersistedMixStatus{
		MixStatus: models.MixStatus{
			IPVersion: "6",
			PubKey:    "pubkey1",
			Up:        &booltrue,
		},
		Timestamp: 1234,
	}

	statuses := []models.PersistedMixStatus{m1, m2}
	return statuses
}

// XSSMixStatus ...
func XSSMixStatus() models.MixStatus {
	booltrue := true
	xss := models.MixStatus{
		IPVersion: "6",
		PubKey:    "pubkey2<script>alert('gotcha')</script>",
		Up:        &booltrue,
	}
	return xss
}

// GoodMixStatus ...
func GoodMixStatus() models.MixStatus {
	booltrue := true
	return models.MixStatus{
		IPVersion: "6",
		PubKey:    "pubkey2",
		Up:        &booltrue,
	}
}

// XSSBatchMixStatus ...
func XSSBatchMixStatus() models.BatchMixStatus {
	booltrue := true
	xss := models.BatchMixStatus{
		Status: []models.MixStatus{
			{
				IPVersion: "6",
				PubKey:    "pubkey2<script>alert('gotcha')</script>",
				Up:        &booltrue,
			},
			{
				IPVersion: "4",
				PubKey:    "pubkey2<script>alert('gotcha')</script>",
				Up:        &booltrue,
			},
			{
				IPVersion: "6",
				PubKey:    "pubkey3<script>alert('gotcha')</script>",
				Up:        &booltrue,
			},
		},
	}
	return xss
}

// GoodBatchMixStatus ...
func GoodBatchMixStatus() models.BatchMixStatus {
	booltrue := true
	return models.BatchMixStatus{
		Status: []models.MixStatus{
			{
				IPVersion: "6",
				PubKey:    "pubkey2",
				Up:        &booltrue,
			},
			{
				IPVersion: "4",
				PubKey:    "pubkey2",
				Up:        &booltrue,
			},
			{
				IPVersion: "6",
				PubKey:    "pubkey3",
				Up:        &booltrue,
			},
		},
	}
}

// GoodPersistedMixStatus ...
func GoodPersistedMixStatus() models.PersistedMixStatus {
	return models.PersistedMixStatus{
		MixStatus: GoodMixStatus(),
		Timestamp: 1234,
	}
}

// GoodPersistedBatchMixStatus ...
func GoodPersistedBatchMixStatus() []models.PersistedMixStatus {
	mixStatus := GoodBatchMixStatus()
	persisted := make([]models.PersistedMixStatus, len(mixStatus.Status))
	for i, status := range mixStatus.Status {
		persisted[i] = models.PersistedMixStatus{
			MixStatus: status,
			Timestamp: 1234,
		}
	}
	return persisted
}

// MixStatusReport ...
func MixStatusReport() models.MixStatusReport {
	return models.MixStatusReport{
		PubKey:           "key1",
		MostRecentIPV4:   true,
		Last5MinutesIPV4: 100,
		LastHourIPV4:     100,
		LastDayIPV4:      100,
		MostRecentIPV6:   true,
		Last5MinutesIPV6: 100,
		LastHourIPV6:     100,
		LastDayIPV6:      100,
	}
}

func GoodMixRegistrationInfo() models.MixRegistrationInfo {
	return models.MixRegistrationInfo{
		NodeInfo: models.NodeInfo{
			MixHost:     "1.2.3.4:1789",
			IdentityKey: "D6YaMzLSY7mANtSQRKXsmMZpqgqiVkeiagKM4V4oFPFr",
			SphinxKey:   "51j2kyqE7iTYc8RBtn5FR5E9jp8BdqZamggSg4PYN6ie",
			Version:     "0.9.2",
			Location:    "London, UK",
		},
		Layer: 1,
	}
}

func GoodGatewayRegistrationInfo() models.GatewayRegistrationInfo {
	return models.GatewayRegistrationInfo{
		NodeInfo: models.NodeInfo{
			MixHost:     "5.6.7.8:1789",
			IdentityKey: "3ebjp1Fb9hdcS1AR6AZihgeJiMHkB5jjJUsvqNnfQwU7",
			SphinxKey:   "E7eRtfBY2TeaREfdk6Wua9pXMrbWsJNEQyNaboa9Yx26",
			Version:     "0.9.2",
			Location:    "Neuchatel, CH",
		},
		ClientsHost: "ws://5.6.7.8:9000",
	}
}

func GoodRegisteredMix() models.RegisteredMix {
	return models.RegisteredMix{
		MixRegistrationInfo: GoodMixRegistrationInfo(),
	}
}

func GoodRegisteredGateway() models.RegisteredGateway {
	return models.RegisteredGateway{
		GatewayRegistrationInfo: GoodGatewayRegistrationInfo(),
	}
}
