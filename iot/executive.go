// Copyright 2019,2020 Alan Tracey Wootton
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package iot

// Utility and controller struct and functions for LookupTable

// Executive is
type Executive struct {
	Looker *LookupTableStruct
	Config *ContactStructConfig
}

// NewExecutive A wrapper to hold and operate a  and the c
func NewExecutive(sizeEstimate int, aname string) *Executive {

	look0 := NewLookupTable(sizeEstimate)
	config0 := NewContactStructConfig(look0)
	config0.Name = aname

	e := Executive{}
	e.Looker = look0
	e.Config = config0
	return &e

}
