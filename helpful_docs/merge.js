[
  // Step 0:
  {
    $match: {
      siteId: 1,
      channelId: 1
    }
  },
  // Step 1: Match relevant documents first to reduce data volume early
  {
    $match: {
      $or: [
        {
          startTimestamp: {
            $gte: 1732271925859,
            $lte: 1735672214979
          }
        },
        {
          endTimestamp: {
            $gte: 1732271925859,
            $lte: 1735672214979
          }
        },
        {
          $and: [
            {
              startTimestamp: {
                $lte: 1732271925859
              }
            },
            {
              endTimestamp: {
                $gte: 1735672214979
              }
            }
          ]
        }
      ]
    }
  },
  // Step 2: Sort matched documents
  {
    $sort: {
      startTimestamp: 1
    }
  },
  // Step 3: Add effectiveEndTimestamp field
  {
    $addFields: {
      effectiveEndTimestamp: {
        $add: ["$endTimestamp", 5000]
      } // Fixed timeOffset = 5000
    }
  },
  // Step 4: Use $setWindowFields for calculating boundaries
  {
    $setWindowFields: {
      sortBy: {
        startTimestamp: 1
      },
      output: {
        prevEffectiveEndTimestamp: {
          $shift: {
            output: "$effectiveEndTimestamp",
            by: -1
          }
        }
      }
    }
  },
  // Step 5: $set boundary value
  {
    $set: {
      boundary: {
        $sum: {
          $cond: [
            {
              $or: [
                {
                  $eq: [
                    "$prevEffectiveEndTimestamp",
                    null
                  ]
                },
                // First document
                {
                  $lt: [
                    "$prevEffectiveEndTimestamp",
                    "$startTimestamp"
                  ]
                } // New group boundary
              ]
            },
            1,
            0
          ]
        }
      }
    }
  },
  // Step 6: Use $setWindowFields for calculating groupId
  {
    $setWindowFields: {
      sortBy: {
        startTimestamp: 1
      },
      output: {
        groupId: {
          $sum: "$boundary",
          window: {
            documents: ["unbounded", "current"]
          }
        }
      }
    }
  },
  // Step 7: Group documents by groupId for merging
  {
    $group: {
      _id: "$groupId",
      startTimestamp: {
        $first: "$startTimestamp"
      },
      endTimestamp: {
        $last: "$endTimestamp"
      },
      objectCount: {
        $sum: "$objectCount"
      }
    }
  },
  // Step 8: Sort merged results by startTimestamp (optional)
  {
    $sort: {
      startTimestamp: 1
    }
  }
]